package tlog

import (
	"context"
	"encoding"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/tidwall/pretty"
	"github.com/tigorhutasuhut/telemetry-go/tcaller"
)

type StderrHandlerOptions struct {
	Level slog.Leveler
	IsTTY *bool
}

type stderrHandler struct {
	writer io.Writer
	level  slog.Leveler
	isTTY  bool
	json   slog.Handler
	attrs  []slog.Attr
	groups []string
}

// NewStderrHandler creates a stderr-oriented slog handler.
//
// TTY output uses a compact colored text format. Non-TTY output falls back to
// JSON output close to slog.JSONHandler.
func NewStderrHandler(w io.Writer, opts *StderrHandlerOptions) slog.Handler {
	var level slog.Leveler = slog.LevelInfo
	if opts != nil && opts.Level != nil {
		level = opts.Level
	}

	isTTY := detectTTY(w)
	if opts != nil && opts.IsTTY != nil {
		isTTY = *opts.IsTTY
	}

	h := &stderrHandler{writer: w, level: level, isTTY: isTTY}
	if !isTTY {
		h.json = slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level, AddSource: true})
	}
	return h
}

// NewLogger creates a logger using the custom stderr handler.
func NewLogger(w io.Writer, opts *StderrHandlerOptions) *slog.Logger {
	return slog.New(NewStderrHandler(w, opts))
}

func (h *stderrHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if h.json != nil {
		return h.json.Enabled(ctx, level)
	}
	return level >= h.level.Level()
}

func (h *stderrHandler) Handle(ctx context.Context, record slog.Record) error {
	if h.json != nil {
		clone := record.Clone()
		for _, attr := range h.attrs {
			clone.AddAttrs(applyGroups(h.groups, attr))
		}
		return h.json.Handle(ctx, clone)
	}

	data := map[string]any{}
	for _, attr := range h.attrs {
		writeAttr(data, h.groups, attr)
	}
	record.Attrs(func(attr slog.Attr) bool {
		writeAttr(data, h.groups, attr)
		return true
	})

	var b strings.Builder
	b.WriteString("[")
	b.WriteString(record.Time.Format("15:04:05.000"))
	b.WriteString("] ")
	b.WriteString(colorLevel(record.Level))

	if record.PC != 0 {
		caller := tcaller.FromPC(record.PC)
		if fileLine := caller.FileLine(); fileLine != "unknown" {
			b.WriteString(" <")
			b.WriteString(fileLine)
			b.WriteString(">")
		}
	}

	if record.Message != "" {
		b.WriteString(" ")
		b.WriteString(record.Message)
	}

	if len(data) > 0 {
		if details := formatTTYDetails(data); len(details) > 0 {
			b.WriteString("\n")
			b.Write(details)
		}
	}
	b.WriteString("\n\n")

	_, err := io.WriteString(h.writer, b.String())
	return err
}

func (h *stderrHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if h.json != nil {
		return &stderrHandler{writer: h.writer, level: h.level, isTTY: h.isTTY, json: h.json.WithAttrs(attrs)}
	}
	clone := *h
	clone.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &clone
}

func (h *stderrHandler) WithGroup(name string) slog.Handler {
	if h.json != nil {
		return &stderrHandler{writer: h.writer, level: h.level, isTTY: h.isTTY, json: h.json.WithGroup(name)}
	}
	clone := *h
	clone.groups = append(append([]string{}, h.groups...), name)
	return &clone
}

func applyGroups(groups []string, attr slog.Attr) slog.Attr {
	if len(groups) == 0 {
		return attr
	}
	wrapped := attr
	for i := len(groups) - 1; i >= 0; i-- {
		wrapped = slog.Group(groups[i], wrapped)
	}
	return wrapped
}

func writeAttr(dst map[string]any, groups []string, attr slog.Attr) {
	attr = applyGroups(groups, attr)
	writeResolvedAttr(dst, attr)
}

func writeResolvedAttr(dst map[string]any, attr slog.Attr) {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return
	}
	if attr.Key == "" && attr.Value.Kind() == slog.KindGroup {
		for _, nested := range attr.Value.Group() {
			writeResolvedAttr(dst, nested)
		}
		return
	}
	if attr.Value.Kind() == slog.KindGroup {
		group := map[string]any{}
		for _, nested := range attr.Value.Group() {
			writeResolvedAttr(group, nested)
		}
		dst[attr.Key] = group
		return
	}
	dst[attr.Key] = normalizeValue(attr.Value)
}

func normalizeValue(v slog.Value) any {
	v = v.Resolve()
	switch v.Kind() {
	case slog.KindString:
		return v.String()
	case slog.KindInt64:
		return v.Int64()
	case slog.KindUint64:
		return v.Uint64()
	case slog.KindFloat64:
		return v.Float64()
	case slog.KindBool:
		return v.Bool()
	case slog.KindDuration:
		return v.Duration().String()
	case slog.KindTime:
		return v.Time()
	case slog.KindGroup:
		m := map[string]any{}
		for _, attr := range v.Group() {
			writeResolvedAttr(m, attr)
		}
		return m
	case slog.KindAny:
		return normalizeAny(v.Any())
	default:
		return v.Any()
	}
}

func normalizeAny(v any) any {
	switch x := v.(type) {
	case nil:
		return nil
	case json.RawMessage:
		if json.Valid(x) {
			return x
		}
		return string(x)
	case []byte:
		if json.Valid(x) {
			return json.RawMessage(x)
		}
		return string(x)
	case time.Duration:
		return x.String()
	case encoding.TextMarshaler:
		text, err := x.MarshalText()
		if err != nil {
			return "ERROR!" + err.Error()
		}
		return string(text)
	case json.Marshaler:
		b, err := x.MarshalJSON()
		if err != nil {
			return "ERROR!" + err.Error()
		}
		if json.Valid(b) {
			return json.RawMessage(b)
		}
		return string(b)
	default:
		return x
	}
}

func formatTTYDetails(data map[string]any) []byte {
	encoded, err := json.Marshal(data)
	if err != nil {
		encoded = []byte(`{"error":"` + err.Error() + `"}`)
	}
	if string(encoded) == "{}" {
		return nil
	}
	formatted := pretty.PrettyOptions(encoded, &pretty.Options{
		Width:    80,
		Prefix:   "",
		Indent:   "  ",
		SortKeys: false,
	})
	return pretty.Color(formatted, nil)
}

func detectTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func colorLevel(level slog.Level) string {
	label := strings.ToUpper(level.String())
	switch {
	case level <= slog.LevelDebug:
		return "\x1b[36m[" + label + "]\x1b[0m"
	case level < slog.LevelWarn:
		return "\x1b[32m[" + label + "]\x1b[0m"
	case level < slog.LevelError:
		return "\x1b[33m[" + label + "]\x1b[0m"
	default:
		return "\x1b[31m[" + label + "]\x1b[0m"
	}
}
