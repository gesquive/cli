package cli

import (
	"context"
	"encoding"
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/mattn/go-colorable"
)

type cliColor string

// These consts are based off of
//  https://gist.github.com/JBlond/2fea43a3049b38287e5e9cefc87b2124

// Base attributes
const (
	cliReset        = cliColor("\033[0m")
	cliBold         = cliColor("\033[1m")
	cliFaint        = cliColor("\033[2m")
	cliItalic       = cliColor("\033[3m")
	cliUnderline    = cliColor("\033[4m")
	cliBlinkSlow    = cliColor("\033[5m")
	cliBlinkRapid   = cliColor("\033[6m")
	cliReverseVideo = cliColor("\033[7m")
	cliConcealed    = cliColor("\033[8m")
	cliCrossedOut   = cliColor("\033[9m")
)

// Foreground text colors
const (
	cliFgBlack   = cliColor("\033[30m")
	cliFgRed     = cliColor("\033[31m")
	cliFgGreen   = cliColor("\033[32m")
	cliFgYellow  = cliColor("\033[33m")
	cliFgBlue    = cliColor("\033[34m")
	cliFgMagenta = cliColor("\033[35m")
	cliFgCyan    = cliColor("\033[36m")
	cliFgWhite   = cliColor("\033[37m")
)

// Foreground Hi-Intensity text colors
const (
	cliFgHiBlack   = cliColor("\033[90m")
	cliFgHiRed     = cliColor("\033[91m")
	cliFgHiGreen   = cliColor("\033[92m")
	cliFgHiYellow  = cliColor("\033[93m")
	cliFgHiBlue    = cliColor("\033[94m")
	cliFgHiMagenta = cliColor("\033[95m")
	cliFgHiCyan    = cliColor("\033[96m")
	cliFgHiWhite   = cliColor("\033[97m")
)

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
	TimeFormat string

	// Disable color (Default: false)
	NoColor bool
}

var defaultLevel = slog.LevelInfo
var defaultTimeFormat = time.DateTime

type Handler struct {
	h      slog.Handler
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

func NewHandler(w io.Writer, opts *HandlerOptions) slog.Handler {
	f, hasFd := w.(*os.File)
	if hasFd {
		w = colorable.NewColorable(f)
	}

	if opts == nil {
		opts = &HandlerOptions{}
	}
	h := &Handler{
		h: slog.NewTextHandler(w, &slog.HandlerOptions{
			AddSource:   opts.AddSource,
			Level:       opts.Level,
			ReplaceAttr: opts.ReplaceAttr,
		}),
		logger:      log.New(w, "", 0),
		addSource:   opts.AddSource,
		level:       defaultLevel,
		replaceAttr: opts.ReplaceAttr,
		timeFormat:  defaultTimeFormat,
		noColor:     opts.NoColor,
	}

	if opts.Level != nil {
		h.level = opts.Level
	}
	if opts.TimeFormat != "" {
		h.timeFormat = opts.TimeFormat
	}

	return h
}

func (h *Handler) clone() *Handler {
	return &Handler{
		logger:      log.New(h.logger.Writer(), "", 0),
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

func SetAsDefault(w io.Writer, opts *HandlerOptions) {
	handler := NewHandler(w, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level.Level()
}

func (h *Handler) SetLogLoggerLevel(level slog.Level) {
	h.level = level
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	buf := newBuffer()
	defer buf.Free()

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
			h.appendAttr(buf, slog.Time(slog.TimeKey, val), h.groupPrefix, nil)
		}
	}

	// level
	if rep == nil {
		h.appendLevel(buf, r.Level)
		buf.WriteByte(' ')
	} else {
		h.appendAttr(buf, slog.Any(slog.LevelKey, r.Level), h.groupPrefix, nil)
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
				h.appendSource(buf, src)
				buf.WriteByte(' ')
			} else {
				h.appendAttr(buf, slog.Any(slog.SourceKey, src), h.groupPrefix, nil)
			}
		}
	}

	// message
	if rep == nil {
		buf.WriteString(r.Message)
		buf.WriteByte(' ')
	} else {
		h.appendAttr(buf, slog.String(slog.MessageKey, r.Message), h.groupPrefix, nil)
	}

	// handler attributes
	if len(h.attrsPrefix) > 0 {
		buf.WriteString(h.attrsPrefix)
	}

	// attributes
	if r.NumAttrs() > 0 {
		r.Attrs(func(attr slog.Attr) bool {
			h.appendAttr(buf, attr, h.groupPrefix, h.groups)
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

	buf := newBuffer()
	defer buf.Free()

	// write attributes to buffer
	for _, attr := range attrs {
		h2.appendAttr(buf, attr, h2.groupPrefix, h2.groups)
	}
	h2.attrsPrefix = h.attrsPrefix + buf.String()
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

func (h *Handler) appendLevel(buf *buffer, level slog.Level) {
	switch level {
	case slog.LevelDebug:
		h.appendANSI(buf, cliFgBlue)
		buf.WriteString("DEBUG")
		h.appendANSI(buf, cliReset)
	case slog.LevelInfo:
		buf.WriteString(" INFO")
	case slog.LevelWarn:
		h.appendANSI(buf, cliFgYellow)
		buf.WriteString(" WARN")
		h.appendANSI(buf, cliReset)
	case slog.LevelError:
		h.appendANSI(buf, cliFgRed)
		buf.WriteString("ERROR")
		h.appendANSI(buf, cliReset)
	default:
		buf.WriteString(level.String())
	}
}

func (h *Handler) appendAttr(buf *buffer, attr slog.Attr, groupsPrefix string, groups []string) {
	if h.replaceAttr != nil && attr.Value.Kind() != slog.KindGroup {
		// Resolve before calling ReplaceAttr, so the user doesn't have to.
		attr.Value = attr.Value.Resolve()
		attr = h.replaceAttr(groups, attr)
	}
	attr.Value = attr.Value.Resolve()

	if attr.Equal(slog.Any("", nil)) {
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

func (h *Handler) appendKey(buf *buffer, key, groups string) {
	h.appendANSI(buf, cliFaint)
	if len(key) == 0 {
		buf.WriteString("\"\"")
	} else {
		appendAutoQuote(buf, groups+key) //TODO: simplify this
	}
	buf.WriteByte('=')
	h.appendANSI(buf, cliReset)
}

func (h *Handler) appendValue(buf *buffer, v slog.Value) {
	switch v.Kind() {
	case slog.KindString:
		appendQuote(buf, v.String())
	case slog.KindInt64:
		buf.Write(strconv.AppendInt(nil, v.Int64(), 10))
	case slog.KindUint64:
		buf.Write(strconv.AppendUint(nil, v.Uint64(), 10))
	case slog.KindFloat64:
		buf.Write(strconv.AppendFloat(nil, v.Float64(), 'g', -1, 64))
	case slog.KindBool:
		buf.Write(strconv.AppendBool(nil, v.Bool()))
	case slog.KindDuration:
		appendQuote(buf, v.Duration().String())
	case slog.KindTime:
		appendQuote(buf, v.Time().String())
	case slog.KindAny:
		switch cv := v.Any().(type) {
		case slog.Level:
			h.appendLevel(buf, cv)
		case encoding.TextMarshaler:
			data, err := cv.MarshalText()
			if err != nil {
				break
			}
			appendQuote(buf, string(data))
		case *slog.Source:
			h.appendSource(buf, cv)
		case []byte:
			appendAutoQuote(buf, string(cv))
		default:
			// Like Printf's %s, we allow both the slice type and the byte element type to be named.
			t := reflect.TypeOf(v.Any())
			if t == nil {
				appendAutoQuote(buf, v.Any().(string))
			} else if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.Uint8 {
				fmt.Fprintf(buf, "\"%s\"", v.Any())
			} else {
				// fmt.Fprint(buf, strconv.Quote(v.Any().(string)))
				fmt.Fprintf(buf, "\"%s\"", v.Any())
			}
		}
	}
}

func (h *Handler) appendError(buf *buffer, err error, attrKey, groupsPrefix string) {
	h.appendANSI(buf, cliFaint)
	h.appendANSI(buf, cliFgRed)
	appendAutoQuote(buf, groupsPrefix+attrKey)
	buf.WriteByte('=')
	h.appendANSI(buf, cliReset)
	appendQuote(buf, err.Error())
}

func (h *Handler) appendSource(buf *buffer, src *slog.Source) {
	dir, file := filepath.Split(src.File)

	h.appendANSI(buf, cliFaint)
	buf.WriteString(filepath.Join(filepath.Base(dir), file))
	buf.WriteByte(':')
	buf.WriteString(strconv.Itoa(src.Line))
	h.appendANSI(buf, cliReset)
}

func (h *Handler) appendANSI(buf *buffer, color cliColor) {
	if !h.noColor {
		buf.WriteString(string(color))
	}
}

// appendString formats using the default formats for its operands and writes to buf.
func appendString(buf *buffer, s string) {
	buf.WriteString(s)
}

// appendQuote wraps the resulting string in quotes
func appendQuote(buf *buffer, s string) {
	*buf = strconv.AppendQuote(*buf, s)
}

// appendAutoQuote will append a string with quotes if the string has spaces, quotes,
// or unprintable characters
func appendAutoQuote(buf *buffer, s string) {
	if needsQuotes(s) {
		appendQuote(buf, s)
	} else {
		appendString(buf, s)
	}
}

func needsQuotes(s string) bool {
	if len(s) == 0 {
		return true
	}

	for _, r := range s {
		switch r {
		case ' ', '"', '=', '\t', '\n', '\v', '\f', '\r', 0x85, 0xA0:
			return true
		}
		if !unicode.IsPrint(r) {
			return true
		}
	}
	return false
}
