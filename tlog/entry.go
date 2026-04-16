package tlog

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/tigorhutasuhut/telemetry-go/tcaller"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type LogOption interface{ applyLog(*Entry) }

type LogOptions = Entry

type Entry struct {
	logger *slog.Logger
	level  slog.Leveler
	msg    string
	fields []any
	caller *tcaller.Caller
	time   *time.Time
}

// Opt creates a logging options builder for Log.
func Opt() *LogOptions { return &LogOptions{} }

// New creates an entry builder for explicit entry-style logging.
func New() *Entry { return &Entry{} }

func (e *Entry) applyLog(dst *Entry) {
	if e == nil {
		return
	}
	if e.level != nil {
		dst.level = e.level
	}
	if e.logger != nil {
		dst.logger = e.logger
	}
	if e.msg != "" {
		dst.msg = e.msg
	}
	if len(e.fields) > 0 {
		dst.fields = append(dst.fields, e.fields...)
	}
	if e.caller != nil {
		caller := *e.caller
		dst.caller = &caller
	}
	if e.time != nil {
		ts := *e.time
		dst.time = &ts
	}
}

// Logger sets the logger used to emit the record.
func (e *Entry) Logger(logger *slog.Logger) *Entry {
	e.logger = logger
	return e
}

// Level sets the log level.
func (e *Entry) Level(level slog.Leveler) *Entry {
	e.level = level
	return e
}

// Message sets the log message. If args are provided it uses fmt.Sprintf.
func (e *Entry) Message(format string, args ...any) *Entry {
	if len(args) > 0 {
		e.msg = fmt.Sprintf(format, args...)
	} else {
		e.msg = format
	}
	return e
}

func (e *Entry) setMessage(msg string) *Entry {
	e.msg = msg
	return e
}

// Fields appends structured log fields.
func (e *Entry) Fields(fields ...any) *Entry {
	e.fields = append(e.fields, fields...)
	return e
}

// Caller sets the caller used for the source field.
func (e *Entry) Caller(caller tcaller.Caller) *Entry {
	e.caller = &caller
	return e
}

// Time sets the record time.
func (e *Entry) Time(ts time.Time) *Entry {
	e.time = &ts
	return e
}

// Log emits the entry.
func (e *Entry) Log(ctx context.Context) { e.Print(ctx) }

// Print emits the entry through the configured slog logger.
func (e *Entry) Print(ctx context.Context) {
	if ctx == nil {
		ctx = context.Background()
	}

	entry := &Entry{}
	if e != nil {
		e.applyLog(entry)
	}
	entry.applyDefaults(ctx, tcaller.New(1))
	if entry.msg == "" && len(entry.fields) == 0 {
		return
	}

	var pc uintptr
	if entry.caller != nil {
		pc = entry.caller.Uintptr()
		if pc > 0 {
			// runtime caller PCs commonly point at the return address. Subtracting
			// 1 keeps the PC inside the calling instruction so slog source
			// resolution maps back to the expected file/function/line.
			pc--
		}
	}

	record := slog.NewRecord(*entry.time, entry.level.Level(), entry.msg, pc)
	for _, attr := range AttrsFromContext(ctx) {
		record.AddAttrs(attr)
	}
	for _, attr := range traceAttrs(ctx) {
		record.AddAttrs(attr)
	}
	if len(entry.fields) > 0 {
		record.Add(entry.fields...)
	}
	logger := entry.logger
	if logger == nil {
		logger = slog.Default()
	}
	_ = logger.Handler().Handle(ctx, record)
}

func (e *Entry) applyDefaults(ctx context.Context, caller tcaller.Caller) {
	if e.level == nil {
		e.level = slog.LevelInfo
	}
	if e.time == nil {
		now := time.Now()
		e.time = &now
	}
	if e.caller == nil {
		if ctxCaller, ok := CallerFromContext(ctx); ok {
			e.caller = &ctxCaller
		} else {
			e.caller = &caller
		}
	}
}

func traceAttrs(ctx context.Context) []slog.Attr {
	if ctx == nil {
		return nil
	}
	sc := oteltrace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return nil
	}
	attrs := make([]slog.Attr, 0, 3)
	attrs = append(attrs,
		slog.String("trace_id", sc.TraceID().String()),
		slog.String("span_id", sc.SpanID().String()),
		slog.String("trace_flags", sc.TraceFlags().String()),
	)
	return attrs
}
