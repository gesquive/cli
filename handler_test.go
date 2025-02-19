package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"runtime"
	"strconv"

	"log/slog"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/exp/slices"
)

const testMessage = "Test logging, but use a somewhat realistic message length."

var (
	testTime     = time.Date(2000, 1, 2, 3, 4, 5, 6, time.UTC)
	testString   = "7e3b3b2aaeff56a7108fe11e154200dd/7819479873059528190"
	testInt      = 32768
	testDuration = 23 * time.Second
	testError    = errors.New("fail")
)

var testAttrs = []slog.Attr{
	slog.String("string", testString),
	slog.Int("status", testInt),
	slog.Duration("duration", testDuration),
	slog.Time("time", testTime),
	slog.Any("error", testError),
}

type TestStruct struct {
	TestTime time.Time
	TestString string
	TestInt int
	TestDuration time.Duration
	TestError error
	testPrivateString string
}

// The next couple of tests are loosely based off of slog/handler_test.go
//  https://cs.opensource.google/go/go/+/master:src/log/slog/handler_test.go

func TestAttrs(t *testing.T) {
	ctx := context.Background()
	preAttrs := []slog.Attr{slog.Int("pre", 0)}
	attrs := []slog.Attr{slog.Int("a", 1), slog.String("b", "two")}
	for _, test := range []struct {
		name  string
		with  func(slog.Handler) slog.Handler
		attrs []slog.Attr
		want  string
	}{
		{
			name: "no attrs",
			want: " INFO message",
		},
		{
			name:  "attrs",
			attrs: attrs,
			want:  " INFO message a=1 b=\"two\"",
		},
		{
			name:  "preformatted",
			with:  func(h slog.Handler) slog.Handler { return h.WithAttrs(preAttrs) },
			attrs: attrs,
			want:  " INFO message pre=0 a=1 b=\"two\"",
		},
		{
			name: "groups",
			attrs: []slog.Attr{
				slog.Int("a", 1),
				slog.Group("g",
					slog.Int("b", 2),
					slog.Group("h", slog.Int("c", 3)),
					slog.Int("d", 4)),
				slog.Int("e", 5),
			},
			want: " INFO message a=1 g.b=2 g.h.c=3 g.d=4 e=5",
		},
		{
			name:  "group",
			with:  func(h slog.Handler) slog.Handler { return h.WithAttrs(preAttrs).WithGroup("s") },
			attrs: attrs,
			want:  " INFO message pre=0 s.a=1 s.b=\"two\"",
		},
		{
			name: "preformatted groups",
			with: func(h slog.Handler) slog.Handler {
				return h.WithAttrs([]slog.Attr{slog.Int("p1", 1)}).
					WithGroup("s1").
					WithAttrs([]slog.Attr{slog.Int("p2", 2)}).
					WithGroup("s2")
			},
			attrs: attrs,
			want:  " INFO message p1=1 s1.p2=2 s1.s2.a=1 s1.s2.b=\"two\"",
		},
		{
			name: "two with-groups",
			with: func(h slog.Handler) slog.Handler {
				return h.WithAttrs([]slog.Attr{slog.Int("p1", 1)}).
					WithGroup("s1").
					WithGroup("s2")
			},
			attrs: attrs,
			want:  " INFO message p1=1 s1.s2.a=1 s1.s2.b=\"two\"",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var buf strings.Builder
			var h slog.Handler = NewHandler(&buf, &HandlerOptions{NoColor: true})
			if test.with != nil {
				h = test.with(h)
			}
			r := slog.NewRecord(time.Time{}, slog.LevelInfo, "message", 0)
			r.AddAttrs(test.attrs...)
			if err := h.Handle(ctx, r); err != nil {
				t.Fatal(err)
			}
			got := strings.TrimRight(buf.String(), "\n")
			if got != test.want {
				t.Errorf("\ngot  %s\nwant %s", got, test.want)
			}
		})
	}
}

func TestCLIHandler(t *testing.T) {
	ctx := context.Background()

	// remove all Attrs
	removeAll := func(_ []string, a slog.Attr) slog.Attr { return slog.Attr{} }

	attrs := []slog.Attr{slog.String("a", "one"), slog.Int("b", 2), slog.Any("", nil)}
	preAttrs := []slog.Attr{slog.Int("pre", 3), slog.String("x", "y")}

	for _, test := range []struct {
		name      string
		replace   func([]string, slog.Attr) slog.Attr
		addSource bool
		with      func(slog.Handler) slog.Handler
		preAttrs  []slog.Attr
		attrs     []slog.Attr
		wantText  string
	}{
		{
			name:     "basic",
			attrs:    attrs,
			wantText: "2000-01-02 03:04:05  INFO message a=\"one\" b=2",
		},
		{
			name:     "empty key",
			attrs:    append(slices.Clip(attrs), slog.Any("", "v")),
			wantText: `2000-01-02 03:04:05  INFO message a="one" b=2 ""="v"`,
		},
		{
			name:     "cap keys",
			replace:  upperCaseKey,
			attrs:    attrs,
			wantText: "2000-01-02 03:04:05  INFO message A=\"one\" B=2",
		},
		{
			name:     "remove all",
			replace:  removeAll,
			attrs:    attrs,
			wantText: "",
		},
		{
			name:     "attrs",
			attrs:    testAttrs,
			wantText: "2000-01-02 03:04:05  INFO message string=\"7e3b3b2aaeff56a7108fe11e154200dd/7819479873059528190\" status=32768 duration=\"23s\" time=\"2000-01-02 03:04:05.000000006 +0000 UTC\" error=\"fail\"",
		},
		{
			name:     "preformatted",
			with:     func(h slog.Handler) slog.Handler { return h.WithAttrs(preAttrs) },
			preAttrs: preAttrs,
			attrs:    attrs,
			wantText: "2000-01-02 03:04:05  INFO message pre=3 x=\"y\" a=\"one\" b=2",
		},
		{
			name:     "preformatted cap keys",
			replace:  upperCaseKey,
			with:     func(h slog.Handler) slog.Handler { return h.WithAttrs(preAttrs) },
			preAttrs: preAttrs,
			attrs:    attrs,
			wantText: "2000-01-02 03:04:05  INFO message PRE=3 X=\"y\" A=\"one\" B=2",
		},
		{
			name:     "preformatted remove all",
			replace:  removeAll,
			with:     func(h slog.Handler) slog.Handler { return h.WithAttrs(preAttrs) },
			preAttrs: preAttrs,
			attrs:    attrs,
			wantText: "",
		},
		{
			name:     "remove built-in",
			replace:  removeKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
			attrs:    attrs,
			wantText: "a=\"one\" b=2",
		},
		{
			name:     "preformatted remove built-in",
			replace:  removeKeys(slog.TimeKey, slog.LevelKey, slog.MessageKey),
			with:     func(h slog.Handler) slog.Handler { return h.WithAttrs(preAttrs) },
			attrs:    attrs,
			wantText: "pre=3 x=\"y\" a=\"one\" b=2",
		},
		{
			name:    "groups",
			replace: removeKeys(slog.TimeKey, slog.LevelKey), // to simplify the result
			attrs: []slog.Attr{
				slog.Int("a", 1),
				slog.Group("g",
					slog.Int("b", 2),
					slog.Group("h", slog.Int("c", 3)),
					slog.Int("d", 4)),
				slog.Int("e", 5),
			},
			wantText: "message a=1 g.b=2 g.h.c=3 g.d=4 e=5",
		},
		{
			name:     "empty group",
			replace:  removeKeys(slog.TimeKey, slog.LevelKey),
			attrs:    []slog.Attr{slog.Group("g"), slog.Group("h", slog.Int("a", 1))},
			wantText: "message h.a=1",
		},
		{
			name:    "escapes",
			replace: removeKeys(slog.TimeKey, slog.LevelKey),
			attrs: []slog.Attr{
				slog.String("a b", "x\t\n\000y"),
				slog.Group(" b.c=\"\\x2E\t",
					slog.String("d=e", "f.g\""),
					slog.Int("m.d", 1)), // dot is not escaped
			},
			wantText: `message "a b"="x\t\n\x00y" " b.c=\"\\x2E\t.d=e"="f.g\"" " b.c=\"\\x2E\t.m.d"=1`,
		},
		{
			name:    "LogValuer",
			replace: removeKeys(slog.TimeKey, slog.LevelKey),
			attrs: []slog.Attr{
				slog.Int("a", 1),
				slog.Any("name", logValueName{"Ren", "Hoek"}),
				slog.Int("b", 2),
			},
			wantText: "message a=1 name.first=\"Ren\" name.last=\"Hoek\" b=2",
		},
		{
			name:     "with-group",
			replace:  removeKeys(slog.TimeKey, slog.LevelKey),
			with:     func(h slog.Handler) slog.Handler { return h.WithAttrs(preAttrs).WithGroup("s") },
			attrs:    attrs,
			wantText: "message pre=3 x=\"y\" s.a=\"one\" s.b=2",
		},
		{
			name:    "preformatted with-groups",
			replace: removeKeys(slog.TimeKey, slog.LevelKey),
			with: func(h slog.Handler) slog.Handler {
				return h.WithAttrs([]slog.Attr{slog.Int("p1", 1)}).
					WithGroup("s1").
					WithAttrs([]slog.Attr{slog.Int("p2", 2)}).
					WithGroup("s2")
			},
			attrs:    attrs,
			wantText: "message p1=1 s1.p2=2 s1.s2.a=\"one\" s1.s2.b=2",
		},
		{
			name:    "two with-groups",
			replace: removeKeys(slog.TimeKey, slog.LevelKey),
			with: func(h slog.Handler) slog.Handler {
				return h.WithAttrs([]slog.Attr{slog.Int("p1", 1)}).
					WithGroup("s1").
					WithGroup("s2")
			},
			attrs:    attrs,
			wantText: "message p1=1 s1.s2.a=\"one\" s1.s2.b=2",
		},
		{
			name:     "GroupValue as Attr value",
			replace:  removeKeys(slog.TimeKey, slog.LevelKey),
			attrs:    []slog.Attr{{"v", slog.AnyValue(slog.IntValue(3))}},
			wantText: "message v=3",
		},
		{
			name:     "byte slice",
			replace:  removeKeys(slog.TimeKey, slog.LevelKey),
			attrs:    []slog.Attr{slog.Any("bs", []byte{1, 2, 3, 4})},
			wantText: `message bs="\x01\x02\x03\x04"`,
		},
		{
			name:     "json.RawMessage",
			replace:  removeKeys(slog.TimeKey, slog.LevelKey),
			attrs:    []slog.Attr{slog.Any("bs", json.RawMessage([]byte("1234")))},
			wantText: `message bs="1234"`,
		},
		{
			name:     "struct",
			replace:  removeKeys(slog.TimeKey, slog.LevelKey),
			attrs:    []slog.Attr{slog.Any("bs", TestStruct{
							TestTime: testTime, 
							TestString: testString,
							TestInt: testInt,
							TestDuration: testDuration,
							TestError: testError,
							testPrivateString: testString,})},
			wantText: `message bs=TestStruct{TestTime=2000-01-02 03:04:05.000000006 +0000 UTC TestString="7e3b3b2aaeff56a7108fe11e154200dd/7819479873059528190" TestInt=32768 TestDuration=23000000000 TestError=fail }`,
		},
		{
			name:    "inline group",
			replace: removeKeys(slog.TimeKey, slog.LevelKey),
			attrs: []slog.Attr{
				slog.Int("a", 1),
				slog.Group("", slog.Int("b", 2), slog.Int("c", 3)),
				slog.Int("d", 4),
			},
			wantText: `message a=1 b=2 c=3 d=4`,
		},
		{
			name: "Source",
			replace: func(gs []string, a slog.Attr) slog.Attr {
				if a.Key == slog.SourceKey {
					s := a.Value.Any().(*slog.Source)
					s.File = filepath.Base(s.File)
					return slog.Any(a.Key, s)
				}
				return removeKeys(slog.TimeKey, slog.LevelKey)(gs, a)
			},
			addSource: true,
			wantText:  `handler_test.go:$LINE message`,
		},
		{
			name:     "errors",
			replace:  removeKeys(slog.TimeKey, slog.LevelKey),
			attrs:    []slog.Attr{slog.Any("err", testError)},
			wantText: `message err="fail"`,
		},
	} {
		r := slog.NewRecord(testTime, slog.LevelInfo, "message", callerPC(2))
		_, _, line, _ := runtime.Caller(0)
		sline := strconv.Itoa(line-1)
		r.AddAttrs(test.attrs...)
		var buf bytes.Buffer
		opts := HandlerOptions{
			ReplaceAttr: test.replace, 
			AddSource: test.addSource,
			NoColor: true,
		}

		t.Run(test.name, func(t *testing.T) {
			h := NewHandler(&buf, &opts)
			if test.with != nil {
				h = test.with(h)
			}
			buf.Reset()
			if err := h.Handle(ctx, r); err != nil {
				t.Fatal(err)
			}
			want := strings.ReplaceAll(test.wantText, "$LINE", sline)
			got := strings.TrimSuffix(buf.String(), "\n")
			if got != want {
				t.Errorf("\ngot  %s\nwant %s\n", got, want)
			}
		})
	}
}

// removeKeys returns a function suitable for HandlerOptions.ReplaceAttr
// that removes all Attrs with the given keys.
func removeKeys(keys ...string) func([]string, slog.Attr) slog.Attr {
	return func(_ []string, a slog.Attr) slog.Attr {
		for _, k := range keys {
			if a.Key == k {
				return slog.Attr{}
			}
		}
		return a
	}
}

func upperCaseKey(_ []string, a slog.Attr) slog.Attr {
	a.Key = strings.ToUpper(a.Key)
	return a
}

type logValueName struct {
	first, last string
}

func (n logValueName) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("first", n.first),
		slog.String("last", n.last))
}

// callerPC returns the program counter at the given stack depth.
func callerPC(depth int) uintptr {
	var pcs [1]uintptr
	runtime.Callers(depth, pcs[:])
	return pcs[0]
}

func TestSecondWith(t *testing.T) {
	// Verify that a second call to Logger.With does not corrupt
	// the original.
	var buf bytes.Buffer
	h := NewHandler(&buf, &HandlerOptions{ReplaceAttr: removeKeys(slog.TimeKey), NoColor: true})
	logger := slog.New(h).With(
		slog.String("app", "playground"),
		slog.String("role", "tester"),
		slog.Int("data_version", 2),
	)
	appLogger := logger.With("type", "log") // this becomes type=met
	_ = logger.With("type", "metric")
	appLogger.Info("foo")
	got := strings.TrimSpace(buf.String())
	want := `INFO foo app="playground" role="tester" data_version=2 type="log"`
	if got != want {
		t.Errorf("\ngot  %s\nwant %s", got, want)
	}
}

func TestReplaceAttrGroups(t *testing.T) {
	// Verify that ReplaceAttr is called with the correct groups.
	type ga struct {
		groups string
		key    string
		val    string
	}

	var got []ga

	h := NewHandler(io.Discard, &HandlerOptions{ReplaceAttr: func(gs []string, a slog.Attr) slog.Attr {
		v := a.Value.String()
		if a.Key == slog.TimeKey {
			v = "<now>"
		}
		got = append(got, ga{strings.Join(gs, ","), a.Key, v})
		return a
	}})
	slog.New(h).
		With(slog.Int("a", 1)).
		WithGroup("g1").
		With(slog.Int("b", 2)).
		WithGroup("g2").
		With(
			slog.Int("c", 3),
			slog.Group("g3", slog.Int("d", 4)),
			slog.Int("e", 5)).
		Info("m",
			slog.Int("f", 6),
			slog.Group("g4", slog.Int("h", 7)),
			slog.Int("i", 8))

	want := []ga{
		{"", "a", "1"},
		{"g1", "b", "2"},
		{"g1,g2", "c", "3"},
		{"g1,g2,g3", "d", "4"},
		{"g1,g2", "e", "5"},
		{"", "time", "<now>"},
		{"", "level", "INFO"},
		{"", "msg", "m"},
		{"g1,g2", "f", "6"},
		{"g1,g2,g4", "h", "7"},
		{"g1,g2", "i", "8"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("\ngot  %v\nwant %v", got, want)
	}
}

// This benchmark is loosly based off of slog/internal/benchmarks/benchmarks_test.go
//  https://cs.opensource.google/go/go/+/master:src/log/slog/internal/benchmarks/benchmarks_test.go


// A disabledHandler's Enabled method always returns false.
type disabledHandler struct{}

func (disabledHandler) Enabled(context.Context, slog.Level) bool  { return false }
func (disabledHandler) Handle(context.Context, slog.Record) error { panic("should not be called") }

func (disabledHandler) WithAttrs([]slog.Attr) slog.Handler {
	panic("disabledHandler: With unimplemented")
}

func (disabledHandler) WithGroup(string) slog.Handler {
	panic("disabledHandler: WithGroup unimplemented")
}

func BenchmarkAttrs(b *testing.B) {
	ctx := context.Background()
	for _, handler := range []struct {
		name     string
		h        slog.Handler
		skipRace bool
	}{
		{"disabled", disabledHandler{}, false},
		{"cli", NewHandler(io.Discard, nil), false},
		{"text", slog.NewTextHandler(io.Discard, nil), false},
		{"json", slog.NewJSONHandler(io.Discard, nil), false},
	} {
		logger := slog.New(handler.h)
		b.Run(handler.name, func(b *testing.B) {
			if handler.skipRace {
				b.Skip("skipping benchmark in race mode")
			}
			for _, call := range []struct {
				name string
				f    func()
			}{
				{
					// The number should match nAttrsInline in slog/record.go.
					// This should exercise the code path where no allocations
					// happen in Record or Attr. If there are allocations, they
					// should only be from Duration.String and Time.String.
					"5 args",
					func() {
						logger.LogAttrs(nil, slog.LevelInfo, testMessage,
							slog.String("string", testString),
							slog.Int("status", testInt),
							slog.Duration("duration", testDuration),
							slog.Time("time", testTime),
							slog.Any("error", testError),
						)
					},
				},
				{
					"5 args ctx",
					func() {
						logger.LogAttrs(ctx, slog.LevelInfo, testMessage,
							slog.String("string", testString),
							slog.Int("status", testInt),
							slog.Duration("duration", testDuration),
							slog.Time("time", testTime),
							slog.Any("error", testError),
						)
					},
				},
				{
					"10 args",
					func() {
						logger.LogAttrs(nil, slog.LevelInfo, testMessage,
							slog.String("string", testString),
							slog.Int("status", testInt),
							slog.Duration("duration", testDuration),
							slog.Time("time", testTime),
							slog.Any("error", testError),
							slog.String("string", testString),
							slog.Int("status", testInt),
							slog.Duration("duration", testDuration),
							slog.Time("time", testTime),
							slog.Any("error", testError),
						)
					},
				},
				{
					// Try an extreme value to see if the results are reasonable.
					"40 args",
					func() {
						logger.LogAttrs(nil, slog.LevelInfo, testMessage,
							slog.String("string", testString),
							slog.Int("status", testInt),
							slog.Duration("duration", testDuration),
							slog.Time("time", testTime),
							slog.Any("error", testError),
							slog.String("string", testString),
							slog.Int("status", testInt),
							slog.Duration("duration", testDuration),
							slog.Time("time", testTime),
							slog.Any("error", testError),
							slog.String("string", testString),
							slog.Int("status", testInt),
							slog.Duration("duration", testDuration),
							slog.Time("time", testTime),
							slog.Any("error", testError),
							slog.String("string", testString),
							slog.Int("status", testInt),
							slog.Duration("duration", testDuration),
							slog.Time("time", testTime),
							slog.Any("error", testError),
							slog.String("string", testString),
							slog.Int("status", testInt),
							slog.Duration("duration", testDuration),
							slog.Time("time", testTime),
							slog.Any("error", testError),
							slog.String("string", testString),
							slog.Int("status", testInt),
							slog.Duration("duration", testDuration),
							slog.Time("time", testTime),
							slog.Any("error", testError),
							slog.String("string", testString),
							slog.Int("status", testInt),
							slog.Duration("duration", testDuration),
							slog.Time("time", testTime),
							slog.Any("error", testError),
							slog.String("string", testString),
							slog.Int("status", testInt),
							slog.Duration("duration", testDuration),
							slog.Time("time", testTime),
							slog.Any("error", testError),
						)
					},
				},
			} {
				b.Run(call.name, func(b *testing.B) {
					b.ReportAllocs()
					b.RunParallel(func(pb *testing.PB) {
						for pb.Next() {
							call.f()
						}
					})
				})
			}
		})
	}
}
