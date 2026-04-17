package tlog

import (
	"context"
	"encoding"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/tigorhutasuhut/telemetry-go/tcaller"
)

const defaultOTLPLoggerName = "github.com/tigorhutasuhut/telemetry-go/tlog"

// OTLPHandlerOptions configures the OTLP slog handler.
type OTLPHandlerOptions struct {
	// Level sets the minimum log level. Defaults to slog.LevelInfo.
	Level slog.Leveler

	// LoggerName is the instrumentation scope name passed to the LoggerProvider.
	// Defaults to "github.com/tigorhutasuhut/telemetry-go/tlog".
	LoggerName string

	// LoggerVersion is the instrumentation scope version.
	LoggerVersion string

	// Provider is the LoggerProvider to use. Defaults to the global provider.
	Provider otellog.LoggerProvider
}

// otlpHandler is an slog.Handler that emits records via the OpenTelemetry Logs API.
//
// Each Handle call converts the slog.Record into an otellog.Record and emits it
// through the configured Logger. Trace correlation is injected as top-level
// fields (TraceId, SpanId, TraceFlags) and also as body attributes so that
// both OTLP-native consumers and attribute-level search work correctly.
//
// Source information from the slog record's PC is mapped to the code.*
// semantic conventions (code.function, code.filepath, code.lineno).
//
// Value normalization follows the plan rules:
//   - time.Duration -> duration.String()
//   - json.RawMessage -> JSON string (OTLP attrs)
//   - []byte valid JSON -> JSON string
//   - json.Marshaler -> JSON string
//   - encoding.TextMarshaler -> text string
//   - unknown any -> json.Marshal; on failure "ERROR!<err>"
type otlpHandler struct {
	logger otellog.Logger
	level  slog.Leveler
	attrs  []slog.Attr
	groups []string
}

// NewOTLPHandler creates an slog.Handler that emits log records via the
// OpenTelemetry Logs API.
//
// If opts.Provider is nil the global LoggerProvider is used. The handler is
// safe to use concurrently.
func NewOTLPHandler(opts *OTLPHandlerOptions) slog.Handler {
	var level slog.Leveler = slog.LevelInfo
	loggerName := defaultOTLPLoggerName
	var loggerVersion string
	var provider otellog.LoggerProvider

	if opts != nil {
		if opts.Level != nil {
			level = opts.Level
		}
		if opts.LoggerName != "" {
			loggerName = opts.LoggerName
		}
		loggerVersion = opts.LoggerVersion
		provider = opts.Provider
	}

	var logger otellog.Logger
	if provider != nil {
		if loggerVersion != "" {
			logger = provider.Logger(loggerName, otellog.WithInstrumentationVersion(loggerVersion))
		} else {
			logger = provider.Logger(loggerName)
		}
	} else {
		if loggerVersion != "" {
			logger = global.Logger(loggerName, otellog.WithInstrumentationVersion(loggerVersion))
		} else {
			logger = global.Logger(loggerName)
		}
	}

	return &otlpHandler{
		logger: logger,
		level:  level,
	}
}

func (h *otlpHandler) Enabled(ctx context.Context, level slog.Level) bool {
	if level < h.level.Level() {
		return false
	}
	param := otellog.EnabledParameters{Severity: slogLevelToOTLP(level)}
	return h.logger.Enabled(ctx, param)
}

func (h *otlpHandler) Handle(ctx context.Context, record slog.Record) error {
	if ctx == nil {
		ctx = context.Background()
	}

	var r otellog.Record

	// Timestamp + observed timestamp.
	r.SetTimestamp(record.Time)
	r.SetObservedTimestamp(time.Now())

	// Severity.
	severity := slogLevelToOTLP(record.Level)
	r.SetSeverity(severity)
	r.SetSeverityText(record.Level.String())

	// Body: the log message.
	if record.Message != "" {
		r.SetBody(otellog.StringValue(record.Message))
	}

	// Trace correlation: top-level fields are set automatically by the OTLP
	// SDK when it reads the span context from ctx in Emit. We also inject the
	// IDs as explicit attributes so attribute-level search works correctly.
	sc := oteltrace.SpanContextFromContext(ctx)
	if sc.IsValid() {
		r.AddAttributes(
			otellog.String("trace_id", sc.TraceID().String()),
			otellog.String("span_id", sc.SpanID().String()),
			otellog.String("trace_flags", sc.TraceFlags().String()),
		)
	}

	// Source information mapped to code.* semantic conventions.
	if record.PC != 0 {
		pc := record.PC - 1 // keep PC inside the calling instruction
		caller := tcaller.FromPC(pc)
		if fn := caller.ShortFunction(); fn != "" {
			r.AddAttributes(otellog.String("code.function", fn))
		}
		if file := caller.File(); file != "" {
			r.AddAttributes(otellog.String("code.filepath", file))
		}
		if line := caller.Line(); line > 0 {
			r.AddAttributes(otellog.Int("code.lineno", line))
		}
	}

	// Handler-level attrs (from WithAttrs calls).
	for _, attr := range h.attrs {
		addSlogAttrToOTLP(&r, h.groups, attr)
	}

	// Record-level attrs.
	record.Attrs(func(attr slog.Attr) bool {
		addSlogAttrToOTLP(&r, h.groups, attr)
		return true
	})

	h.logger.Emit(ctx, r)
	return nil
}

func (h *otlpHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &clone
}

func (h *otlpHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	clone := *h
	clone.groups = append(append([]string{}, h.groups...), name)
	return &clone
}

// ---------------------------------------------------------------------------
// slog attr → OTLP attribute conversion
// ---------------------------------------------------------------------------

func addSlogAttrToOTLP(r *otellog.Record, groups []string, attr slog.Attr) {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return
	}
	kvs := slogAttrToOTLPKVs(groups, attr)
	r.AddAttributes(kvs...)
}

// slogAttrToOTLPKVs converts a single slog.Attr (possibly a group) into a
// flat slice of otellog.KeyValue, respecting the current group stack.
func slogAttrToOTLPKVs(groups []string, attr slog.Attr) []otellog.KeyValue {
	attr.Value = attr.Value.Resolve()
	if attr.Equal(slog.Attr{}) {
		return nil
	}

	if attr.Value.Kind() == slog.KindGroup {
		var kvs []otellog.KeyValue
		nextGroups := groups
		if attr.Key != "" {
			nextGroups = append(append([]string{}, groups...), attr.Key)
		}
		for _, nested := range attr.Value.Group() {
			kvs = append(kvs, slogAttrToOTLPKVs(nextGroups, nested)...)
		}
		return kvs
	}

	key := buildGroupedKey(groups, attr.Key)
	val := slogValueToOTLP(attr.Value)
	return []otellog.KeyValue{otellog.KeyValue{Key: key, Value: val}}
}

func buildGroupedKey(groups []string, key string) string {
	if len(groups) == 0 {
		return key
	}
	out := ""
	for _, g := range groups {
		out += g + "."
	}
	return out + key
}

// slogValueToOTLP converts a resolved slog.Value to an otellog.Value.
// Per plan normalization rules for OTLP attrs:
//   - json.RawMessage -> JSON string
//   - json.Marshaler  -> JSON string
//   - encoding.TextMarshaler -> text string
//   - time.Duration -> duration.String()
//   - []byte valid JSON -> JSON string; otherwise hex/string
//   - unknown any -> json.Marshal; on failure "ERROR!<err>"
func slogValueToOTLP(v slog.Value) otellog.Value {
	v = v.Resolve()
	switch v.Kind() {
	case slog.KindString:
		return otellog.StringValue(v.String())
	case slog.KindBool:
		return otellog.BoolValue(v.Bool())
	case slog.KindInt64:
		return otellog.Int64Value(v.Int64())
	case slog.KindUint64:
		// OTLP doesn't have uint64; use int64 (wraps for very large values).
		return otellog.Int64Value(int64(v.Uint64()))
	case slog.KindFloat64:
		return otellog.Float64Value(v.Float64())
	case slog.KindDuration:
		return otellog.StringValue(v.Duration().String())
	case slog.KindTime:
		return otellog.StringValue(v.Time().Format(time.RFC3339Nano))
	case slog.KindGroup:
		var kvs []otellog.KeyValue
		for _, attr := range v.Group() {
			kvs = append(kvs, otellog.KeyValue{
				Key:   attr.Key,
				Value: slogValueToOTLP(attr.Value),
			})
		}
		return otellog.MapValue(kvs...)
	case slog.KindAny:
		return anyToOTLP(v.Any())
	default:
		return otellog.StringValue(fmt.Sprintf("%v", v.Any()))
	}
}

func anyToOTLP(v any) otellog.Value {
	switch x := v.(type) {
	case nil:
		return otellog.Value{}
	case bool:
		return otellog.BoolValue(x)
	case int:
		return otellog.Int64Value(int64(x))
	case int8:
		return otellog.Int64Value(int64(x))
	case int16:
		return otellog.Int64Value(int64(x))
	case int32:
		return otellog.Int64Value(int64(x))
	case int64:
		return otellog.Int64Value(x)
	case uint:
		return otellog.Int64Value(int64(x))
	case uint8:
		return otellog.Int64Value(int64(x))
	case uint16:
		return otellog.Int64Value(int64(x))
	case uint32:
		return otellog.Int64Value(int64(x))
	case uint64:
		return otellog.Int64Value(int64(x))
	case float32:
		return otellog.Float64Value(float64(x))
	case float64:
		return otellog.Float64Value(x)
	case string:
		return otellog.StringValue(x)
	case time.Duration:
		return otellog.StringValue(x.String())
	case time.Time:
		return otellog.StringValue(x.Format(time.RFC3339Nano))
	case json.RawMessage:
		// OTLP attrs: JSON string representation.
		return otellog.StringValue(string(x))
	case []byte:
		if json.Valid(x) {
			return otellog.StringValue(string(x))
		}
		return otellog.BytesValue(x)
	case json.Marshaler:
		b, err := x.MarshalJSON()
		if err != nil {
			return otellog.StringValue("ERROR!" + err.Error())
		}
		return otellog.StringValue(string(b))
	case encoding.TextMarshaler:
		text, err := x.MarshalText()
		if err != nil {
			return otellog.StringValue("ERROR!" + err.Error())
		}
		return otellog.StringValue(string(text))
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return otellog.StringValue("ERROR!" + err.Error())
		}
		return otellog.StringValue(string(b))
	}
}

// ---------------------------------------------------------------------------
// slog level → OTLP severity
// ---------------------------------------------------------------------------

// slogLevelToOTLP maps slog.Level values to OTLP Severity constants.
//
// slog defines Debug=-4, Info=0, Warn=4, Error=8.
// Values between the named levels are valid; they are mapped to the nearest
// OTLP severity band.
func slogLevelToOTLP(level slog.Level) otellog.Severity {
	switch {
	case level < slog.LevelDebug:
		return otellog.SeverityTrace
	case level < slog.LevelInfo:
		// Debug band: map to Debug1-Debug4 using 1-unit steps.
		offset := int(level - slog.LevelDebug)
		if offset < 0 {
			offset = 0
		} else if offset > 3 {
			offset = 3
		}
		return otellog.SeverityDebug1 + otellog.Severity(offset)
	case level < slog.LevelWarn:
		offset := int(level - slog.LevelInfo)
		if offset < 0 {
			offset = 0
		} else if offset > 3 {
			offset = 3
		}
		return otellog.SeverityInfo1 + otellog.Severity(offset)
	case level < slog.LevelError:
		offset := int(level - slog.LevelWarn)
		if offset < 0 {
			offset = 0
		} else if offset > 3 {
			offset = 3
		}
		return otellog.SeverityWarn1 + otellog.Severity(offset)
	default:
		offset := int(level - slog.LevelError)
		if offset < 0 {
			offset = 0
		} else if offset > 3 {
			offset = 3
		}
		return otellog.SeverityError1 + otellog.Severity(offset)
	}
}
