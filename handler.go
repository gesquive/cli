package cli

import (
	"context"
	"encoding"
	"fmt"
	"io"
	"log"
	"log/slog"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/fatih/color"
)

var fprintStd = color.New().FprintFunc()
var fprintDebug = color.New(color.FgBlue).FprintFunc()
var fprintInfo = color.New().FprintFunc()
var fprintWarn = color.New(color.FgYellow).FprintFunc()
var fprintError = color.New(color.FgRed).FprintFunc()
var fprintAttr = color.New(color.Faint).FprintFunc()
var fprintAttrError = color.New(color.FgRed).Add(color.Faint).FprintFunc()

// HandlerOptions is a drop in replacement for [slog.HandlerOptions]
type HandlerOptions struct {
	// AddSource causes the handler to compute the source code position
	// of the log statement and add a SourceKey attribute to the output.
	AddSource bool

	// Level reports the minimum record level that will be logged.
	// The handler discards records with lower levels.
	// If Level is nil, the handler assumes LevelInfo.
	// The handler calls Level.Level for each record processed;
	// to adjust the minimum level dynamically, use a LevelVar.
	Level slog.Leveler

	// ReplaceAttr is called to rewrite each non-group attribute before it is logged.
	// The attribute's value has been resolved (see [Value.Resolve]).
	// If ReplaceAttr returns a zero Attr, the attribute is discarded.
	//
	// The built-in attributes with keys "time", "level", "source", and "msg"
	// are passed to this function, except that time is omitted
	// if zero, and source is omitted if AddSource is false.
	//
	// The first argument is a list of currently open groups that contain the
	// Attr. It must not be retained or modified. ReplaceAttr is never called
	// for Group attributes, only their contents. For example, the attribute
	// list
	//
	//     Int("a", 1), Group("g", Int("b", 2)), Int("c", 3)
	//
	// results in consecutive calls to ReplaceAttr with the following arguments:
	//
	//     nil, Int("a", 1)
	//     []string{"g"}, Int("b", 2)
	//     nil, Int("c", 3)
	//
	// ReplaceAttr can be used to change the default keys of the built-in
	// attributes, convert types (for example, to replace a `time.Time` with the
	// integer seconds since the Unix epoch), sanitize personal information, or
	// remove attributes from the output.
	ReplaceAttr func(groups []string, a slog.Attr) slog.Attr

	// Time format (Default: time.DateTime)
	TimeFormat  string

	// Disable color (Default: false)
	NoColor     bool   
}

var defaultLevel      = slog.LevelInfo
var defaultTimeFormat = time.DateTime

type Handler struct {
	h slog.Handler
	logger *log.Logger

	attrsPrefix string
	groupPrefix string
	groups      []string

	addSource   bool
	level       slog.Leveler
	replaceAttr func([]string, slog.Attr) slog.Attr
	timeFormat  string
	noColor     bool
}

func NewCLIHandler(w io.Writer, opts *HandlerOptions) slog.Handler {
	if opts == nil {
		opts = &HandlerOptions{}
	}
	h := &Handler{
		h: slog.NewTextHandler(w, &slog.HandlerOptions{
			AddSource: opts.AddSource,
			Level: opts.Level,
			ReplaceAttr: opts.ReplaceAttr,
		}),
		logger: log.New(w, "", 0),
		addSource: opts.AddSource,
		level: defaultLevel,
		replaceAttr: opts.ReplaceAttr,
		timeFormat: defaultTimeFormat,
		noColor: opts.NoColor,
	}

	if opts.Level != nil {
		h.level = opts.Level
	}
	if opts.TimeFormat != "" {
		h.timeFormat = opts.TimeFormat
	}
	color.NoColor = opts.NoColor

	return h
}

func (h *Handler) clone() *Handler {
	return &Handler{
		logger: log.New(h.logger.Writer(), "", 0),
		attrsPrefix: h.attrsPrefix,
		groupPrefix: h.groupPrefix,
		groups:      h.groups,
		addSource:   h.addSource,
		level:       h.level,
		replaceAttr: h.replaceAttr,
		timeFormat:  h.timeFormat,
		noColor:     h.noColor,
	}
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
    return level >= h.level.Level()
}

func (h *Handler) SetLogLoggerLevel(level slog.Level) {
	h.level = level
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	var buf strings.Builder

	// Built-in attributes. They are not in a group.
	// stateGroups := state.groups
	// state.groups = nil // So ReplaceAttrs sees no groups instead of the pre groups.
	rep := h.replaceAttr

	// time
	if !r.Time.IsZero() {
		val := r.Time.Round(0) // strip monotonic to match Attr behavior
		if rep == nil {
			buf.WriteString(r.Time.Format(h.timeFormat))
			buf.WriteByte(' ')
		} else {
			h.appendAttr(&buf, slog.Time(slog.TimeKey, val), h.groupPrefix, nil)
		}
	}

	// level
	if rep == nil {
		h.appendLevel(&buf, r.Level)
		buf.WriteByte(' ')
	} else {
		h.appendAttr(&buf, slog.Any(slog.LevelKey, r.Level), h.groupPrefix, nil)
	}

	// source
	if h.addSource {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		if f.File != "" {
			src := &slog.Source{
				Function: f.Function,
				File:     f.File,
				Line:     f.Line,
			}

			if rep == nil {
				h.appendSource(&buf, src)
				buf.WriteByte(' ')
			} else {
				h.appendAttr(&buf, slog.Any(slog.SourceKey, src), h.groupPrefix, nil)
			}
		}
	}

	// message
	if rep == nil {
		buf.WriteString(r.Message)
		buf.WriteByte(' ')
	} else {
		h.appendAttr(&buf, slog.String(slog.MessageKey, r.Message), h.groupPrefix, nil)
	}
	
	// handler attributes
	if len(h.attrsPrefix) > 0 {
		buf.WriteString(h.attrsPrefix)
	}
	
	// attributes
	if r.NumAttrs() > 0 {
		r.Attrs(func(attr slog.Attr) bool {
			h.appendAttr(&buf, attr, h.groupPrefix, h.groups)
			return true
		})
	}

	h.logger.Println(strings.TrimRight(buf.String(), " "))

	return nil
}

func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	h2 := h.clone()

	var sb strings.Builder

	// write attributes to buffer
	for _, attr := range attrs {
		h2.appendAttr(&sb, attr, h2.groupPrefix, h2.groups)
	}
	h2.attrsPrefix = h.attrsPrefix + sb.String()
	return h2
}

func (h *Handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	h2 := h.clone()
	h2.groupPrefix += name + "."
	h2.groups = append(h2.groups, name)
	return h2
}

func (h *Handler) appendLevel(buf *strings.Builder, level slog.Level) {
	switch level {
	case slog.LevelDebug:
		fprintDebug(buf, level.String())
	case slog.LevelInfo:
		fprintInfo(buf, level.String(), " ")
	case slog.LevelWarn:
		fprintWarn(buf, level.String(), " ")
	case slog.LevelError:
		fprintError(buf, level.String())
	default:
		buf.WriteString(level.String())
	}
}

func (h *Handler) appendAttr(buf *strings.Builder, attr slog.Attr, groupsPrefix string, groups []string) {
	if h.replaceAttr != nil && attr.Value.Kind() != slog.KindGroup {
		// Resolve before calling ReplaceAttr, so the user doesn't have to.
		attr.Value = attr.Value.Resolve()
		attr = h.replaceAttr(groups, attr)
	}
	attr.Value = attr.Value.Resolve()

	// if attr.Equal(slog.Attr{}) || attr.Equal(slog.Any("", nil)) {
	if attr.Key == "" && attr.Value.Any() == nil {
		return
	}

	key := strings.ToLower(attr.Key)
	if attr.Value.Kind() == slog.KindGroup {
		if attr.Key != "" {
			groupsPrefix += attr.Key + "."
			groups = append(groups, attr.Key)
		}
		for _, groupAttr := range attr.Value.Group() {
			h.appendAttr(buf, groupAttr, groupsPrefix, groups)
		}
	} else if key == slog.TimeKey {
		buf.WriteString(attr.Value.Time().Format(h.timeFormat))
		buf.WriteByte(' ')
	} else if key == slog.LevelKey {
		h.appendLevel(buf, attr.Value.Any().(slog.Level))
		buf.WriteByte(' ')
	} else if key == slog.SourceKey {
		h.appendSource(buf, attr.Value.Any().(*slog.Source))
		buf.WriteByte(' ')
	} else if key == slog.MessageKey {
		buf.WriteString(attr.Value.String())
		buf.WriteByte(' ')
	} else if err, ok := attr.Value.Any().(error); ok {
		h.appendError(buf, err, attr.Key, groupsPrefix)
		buf.WriteByte(' ')
	} else {
		h.appendKey(buf, attr.Key, groupsPrefix)
		h.appendValue(buf, attr.Value)
		buf.WriteByte(' ')
	}
}

func (h *Handler) appendKey(buf *strings.Builder, key, groups string) {
	if len(key) == 0 {
		appendString(buf, fprintAttr, "\"\"")
	} else {
		appendAutoQuote(buf, fprintAttr, groups, key)
	}
	appendString(buf, fprintAttr, "=")
}

func (h *Handler) appendValue(buf *strings.Builder, v slog.Value) {
	switch v.Kind() {
	case slog.KindString:
		appendQuote(buf, fprintStd, v.String())
	case slog.KindInt64:
		buf.Write(strconv.AppendInt(nil, v.Int64(), 10))
	case slog.KindUint64:
		buf.Write(strconv.AppendUint(nil, v.Uint64(), 10))
	case slog.KindFloat64:
		buf.Write(strconv.AppendFloat(nil, v.Float64(), 'g', -1, 64))
	case slog.KindBool:
		buf.Write(strconv.AppendBool(nil, v.Bool()))
	case slog.KindDuration:
		appendQuote(buf, fprintStd, v.Duration().String())
	case slog.KindTime:
		appendQuote(buf, fprintStd, v.Time().String())
	case slog.KindAny:
		switch cv := v.Any().(type) {
		case slog.Level:
			h.appendLevel(buf, cv)
		case encoding.TextMarshaler:
			data, err := cv.MarshalText()
			if err != nil {
				break
			}
			appendQuote(buf, fprintStd, string(data))
		case *slog.Source:
			h.appendSource(buf, cv)
		case []byte:
			appendAutoQuote(buf, fprintStd, string(cv))
		default:
			// Like Printf's %s, we allow both the slice type and the byte element type to be named.
			t := reflect.TypeOf(v.Any())
			if t == nil {
				appendAutoQuote(buf, fprintStd, v.Any())
			} else if t.Kind() ==  reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
				fmt.Fprintf(buf, "\"%s\"", v.Any())
			} else {
				// fmt.Fprint(buf, strconv.Quote(v.Any().(string)))
				fmt.Fprintf(buf, "\"%s\"", v.Any())
			}
		}
	}
}

func (h *Handler) appendError(buf *strings.Builder, err error, attrKey, groupsPrefix string) {
	appendAutoQuote(buf, fprintAttrError, groupsPrefix, attrKey)
	appendString(buf, fprintAttrError, "=")
	appendQuote(buf, fprintStd, err.Error())
}

func (h *Handler) appendSource(buf *strings.Builder, src *slog.Source) {
	dir, file := filepath.Split(src.File)
	appendString(buf, fprintAttr, filepath.Join(filepath.Base(dir), file), ":", strconv.Itoa(src.Line))
}

type fPrintFunc func(w io.Writer, a ...interface{}) //(n int, err error)

//appendString formats using the default formats for its operands and writes to buf.
// Spaces are added between operands when neither is a string.
func appendString(buf *strings.Builder, fprint fPrintFunc, a ...interface{}) {
	fprint(buf, a...)
}

//appendQuote wraps the resulting string in quotes
func appendQuote(buf *strings.Builder, fprint fPrintFunc, a ...interface{}) {
	var sb strings.Builder
	prevString := false
	for argNum, arg := range a {
		isString := arg != nil && reflect.TypeOf(arg).Kind() == reflect.String
		// Add a space between two non-string arguments.
		if argNum > 0 && !isString && !prevString {
			fmt.Fprint(&sb, " ")
		}
		fmt.Fprint(&sb, arg.(string))
		prevString = isString
	}
	fprint(buf, strconv.Quote(sb.String()))
}

//appendAutoQuote will append a string with quotes if the string has spaces, quotes,
// or unprintable characters
func appendAutoQuote(buf *strings.Builder, fprint fPrintFunc, a ...interface{}) {
	if needsQuotes(a...) {
		appendQuote(buf, fprint, a...)
	} else {
		appendString(buf, fprint, a...)
	}
}

func needsQuotes(a ...interface{}) bool {
	if len(a) == 0 {
		return true
	}

	prevString := false
	for argNum, arg := range a {
		if arg == nil {
			continue
		}
		isString := reflect.TypeOf(arg).Kind() == reflect.String
		if argNum > 0 && !isString && !prevString {
			return true // appendString would have added a space between these two
		}
		for _, r := range arg.(string) {
			if unicode.IsSpace(r) || r == '"' || r == '=' || !unicode.IsPrint(r) {
				return true
			}
		}
		prevString = isString
	}
	return false


}
