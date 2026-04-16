package telemetry

import (
	"context"
	"time"

	"github.com/tigorhutasuhut/telemetry-go/tcaller"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const defaultTracerName = "github.com/tigorhutasuhut/telemetry-go/telemetry"

// StartSpanOption configures StartSpan.
type StartSpanOption interface{ applySpan(*StartSpanOptions) }

// StartSpanOptions builds tracing options for StartSpan.
//
// When multiple options are passed to StartSpan, later singular values override
// earlier ones, while slice-based values are appended.
type StartSpanOptions struct {
	name          *string
	kind          *trace.SpanKind
	attributes    []attribute.KeyValue
	caller        *tcaller.Caller
	startTime     *time.Time
	startOptions  []trace.SpanStartOption
	tracerName    *string
	tracerVersion *string
	links         []trace.Link
	newRoot       *bool
}

// Span creates a new StartSpanOptions builder.
func Span() *StartSpanOptions { return &StartSpanOptions{} }

func (o *StartSpanOptions) applySpan(cfg *StartSpanOptions) {
	if o == nil {
		return
	}
	if o.name != nil {
		name := *o.name
		cfg.name = &name
	}
	if o.kind != nil {
		kind := *o.kind
		cfg.kind = &kind
	}
	if len(o.attributes) > 0 {
		cfg.attributes = append(cfg.attributes, o.attributes...)
	}
	if o.caller != nil {
		caller := *o.caller
		cfg.caller = &caller
	}
	if o.startTime != nil {
		startTime := *o.startTime
		cfg.startTime = &startTime
	}
	if len(o.startOptions) > 0 {
		cfg.startOptions = append(cfg.startOptions, o.startOptions...)
	}
	if o.tracerName != nil {
		tracerName := *o.tracerName
		cfg.tracerName = &tracerName
	}
	if o.tracerVersion != nil {
		tracerVersion := *o.tracerVersion
		cfg.tracerVersion = &tracerVersion
	}
	if len(o.links) > 0 {
		cfg.links = append(cfg.links, o.links...)
	}
	if o.newRoot != nil {
		newRoot := *o.newRoot
		cfg.newRoot = &newRoot
	}
}

// Name sets the span name.
//
// Span name precedence is: explicit Name, then explicit Caller, then the
// auto-detected caller.
func (o *StartSpanOptions) Name(name string) *StartSpanOptions {
	o.name = &name
	return o
}

// Kind sets the span kind.
func (o *StartSpanOptions) Kind(kind trace.SpanKind) *StartSpanOptions {
	o.kind = &kind
	return o
}

// Attributes appends span attributes.
func (o *StartSpanOptions) Attributes(attrs ...attribute.KeyValue) *StartSpanOptions {
	o.attributes = append(o.attributes, attrs...)
	return o
}

// Caller sets the caller used for default span naming and code.* attributes.
//
// Tracing does not read caller information from context.
func (o *StartSpanOptions) Caller(caller tcaller.Caller) *StartSpanOptions {
	o.caller = &caller
	return o
}

// StartTime sets the span start time.
func (o *StartSpanOptions) StartTime(startTime time.Time) *StartSpanOptions {
	o.startTime = &startTime
	return o
}

// StartOptions appends raw OpenTelemetry span start options.
func (o *StartSpanOptions) StartOptions(opts ...trace.SpanStartOption) *StartSpanOptions {
	o.startOptions = append(o.startOptions, opts...)
	return o
}

// TracerName sets the instrumentation scope name.
func (o *StartSpanOptions) TracerName(name string) *StartSpanOptions {
	o.tracerName = &name
	return o
}

// TracerVersion sets the instrumentation scope version.
func (o *StartSpanOptions) TracerVersion(version string) *StartSpanOptions {
	o.tracerVersion = &version
	return o
}

// Links appends span links.
func (o *StartSpanOptions) Links(links ...trace.Link) *StartSpanOptions {
	o.links = append(o.links, links...)
	return o
}

// NewRoot controls whether the span starts a new trace root.
func (o *StartSpanOptions) NewRoot(newRoot bool) *StartSpanOptions {
	o.newRoot = &newRoot
	return o
}

// StartSpan starts a span using the global OpenTelemetry tracer provider.
//
// Defaults:
//   - span name: short function name of the StartSpan callsite
//   - tracer name: github.com/tigorhutasuhut/telemetry-go/telemetry
//   - caller: stack location of StartSpan's caller
//   - start time: determined by the tracer when not explicitly set
//
// Example:
//
//	ctx, span := telemetry.StartSpan(ctx,
//		telemetry.Span().
//			Name("user.lookup").
//			Attributes(attribute.String("user.id", id)),
//	)
//	defer span.End()
//
// When caller metadata is available, StartSpan also adds these attributes:
//   - code.function
//   - code.filepath
//   - code.lineno
func StartSpan(ctx context.Context, opts ...StartSpanOption) (context.Context, trace.Span) {
	cfg := &StartSpanOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt.applySpan(cfg)
		}
	}

	caller := resolveSpanCaller(cfg.caller)
	name := resolveSpanName(cfg.name, caller)
	startOpts := buildSpanStartOptions(cfg, caller)

	tracerOpts := make([]trace.TracerOption, 0, 1)
	if cfg.tracerVersion != nil && *cfg.tracerVersion != "" {
		tracerOpts = append(tracerOpts, trace.WithInstrumentationVersion(*cfg.tracerVersion))
	}

	tracerName := defaultTracerName
	if cfg.tracerName != nil && *cfg.tracerName != "" {
		tracerName = *cfg.tracerName
	}

	tracer := otel.GetTracerProvider().Tracer(tracerName, tracerOpts...)
	return tracer.Start(ctx, name, startOpts...)
}

func resolveSpanCaller(explicit *tcaller.Caller) tcaller.Caller {
	if explicit != nil {
		return *explicit
	}
	return tcaller.New(2)
}

func resolveSpanName(explicit *string, caller tcaller.Caller) string {
	if explicit != nil && *explicit != "" {
		return *explicit
	}
	if short := caller.ShortFunction(); short != "" {
		return short
	}
	return caller.String()
}

func buildSpanStartOptions(cfg *StartSpanOptions, caller tcaller.Caller) []trace.SpanStartOption {
	startOpts := make([]trace.SpanStartOption, 0, len(cfg.startOptions)+5)
	startOpts = append(startOpts, cfg.startOptions...)

	if cfg.kind != nil {
		startOpts = append(startOpts, trace.WithSpanKind(*cfg.kind))
	}
	if cfg.startTime != nil {
		startOpts = append(startOpts, trace.WithTimestamp(*cfg.startTime))
	}
	if len(cfg.links) > 0 {
		startOpts = append(startOpts, trace.WithLinks(cfg.links...))
	}
	if cfg.newRoot != nil && *cfg.newRoot {
		startOpts = append(startOpts, trace.WithNewRoot())
	}

	attrs := make([]attribute.KeyValue, 0, len(cfg.attributes)+3)
	attrs = append(attrs, cfg.attributes...)
	if function := caller.ShortFunction(); function != "" {
		attrs = append(attrs, attribute.String("code.function", function))
	}
	if file := caller.File(); file != "" {
		attrs = append(attrs, attribute.String("code.filepath", file))
	}
	if line := caller.Line(); line > 0 {
		attrs = append(attrs, attribute.Int("code.lineno", line))
	}
	if len(attrs) > 0 {
		startOpts = append(startOpts, trace.WithAttributes(attrs...))
	}

	return startOpts
}
