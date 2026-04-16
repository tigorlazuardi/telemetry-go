package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/tigorhutasuhut/telemetry-go/tcaller"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/trace"
	tracetest "go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func setupTracerProvider(tb testing.TB) *tracetest.InMemoryExporter {
	tb.Helper()
	exporter := tracetest.NewInMemoryExporter()
	provider := trace.NewTracerProvider(trace.WithSyncer(exporter))
	old := otel.GetTracerProvider()
	otel.SetTracerProvider(provider)
	tb.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
		otel.SetTracerProvider(old)
	})
	return exporter
}

func firstSpan(tb testing.TB, exporter *tracetest.InMemoryExporter) tracetest.SpanStub {
	tb.Helper()
	spans := exporter.GetSpans()
	if len(spans) != 1 {
		tb.Fatalf("span count = %d, want 1", len(spans))
	}
	return spans[0]
}

func startSpanTestCaller() tcaller.Caller {
	return tcaller.Current()
}

func spanAttr(span tracetest.SpanStub, key string) (attribute.Value, bool) {
	for _, attr := range span.Attributes {
		if string(attr.Key) == key {
			return attr.Value, true
		}
	}
	return attribute.Value{}, false
}

func TestStartSpanDefaults(t *testing.T) {
	exporter := setupTracerProvider(t)

	ctx, span := StartSpan(context.Background())
	span.End()
	_ = ctx

	ro := firstSpan(t, exporter)
	if ro.Name != "telemetry.TestStartSpanDefaults" {
		t.Fatalf("Name() = %q", ro.Name)
	}
	if ro.InstrumentationScope.Name != defaultTracerName {
		t.Fatalf("scope name = %q", ro.InstrumentationScope.Name)
	}
	if got, ok := spanAttr(ro, "code.function"); !ok || got.AsString() != "telemetry.TestStartSpanDefaults" {
		t.Fatalf("code.function = %q, ok=%v", got.AsString(), ok)
	}
	if got, ok := spanAttr(ro, "code.filepath"); !ok || got.AsString() == "" {
		t.Fatalf("code.filepath missing")
	}
	if got, ok := spanAttr(ro, "code.lineno"); !ok || got.AsInt64() <= 0 {
		t.Fatalf("code.lineno = %d, ok=%v", got.AsInt64(), ok)
	}
	if ro.StartTime.IsZero() {
		t.Fatal("StartTime() is zero")
	}
}

func TestStartSpanExplicitNameTakesPrecedence(t *testing.T) {
	exporter := setupTracerProvider(t)

	ctx, span := StartSpan(context.Background(), Span().Name("custom.name"))
	span.End()
	_ = ctx

	ro := firstSpan(t, exporter)
	if ro.Name != "custom.name" {
		t.Fatalf("Name() = %q", ro.Name)
	}
}

func TestStartSpanExplicitCallerSetsNameAndCodeAttrs(t *testing.T) {
	exporter := setupTracerProvider(t)
	caller := startSpanTestCaller()

	ctx, span := StartSpan(context.Background(), Span().Caller(caller))
	span.End()
	_ = ctx

	ro := firstSpan(t, exporter)
	if ro.Name != caller.ShortFunction() {
		t.Fatalf("Name() = %q, want %q", ro.Name, caller.ShortFunction())
	}
	if got, ok := spanAttr(ro, "code.function"); !ok || got.AsString() != caller.ShortFunction() {
		t.Fatalf("code.function = %q, want %q", got.AsString(), caller.ShortFunction())
	}
	if got, ok := spanAttr(ro, "code.filepath"); !ok || got.AsString() != caller.File() {
		t.Fatalf("code.filepath = %q, want %q", got.AsString(), caller.File())
	}
	if got, ok := spanAttr(ro, "code.lineno"); !ok || int(got.AsInt64()) != caller.Line() {
		t.Fatalf("code.lineno = %d, want %d", got.AsInt64(), caller.Line())
	}
}

func TestStartSpanAppliesOptions(t *testing.T) {
	exporter := setupTracerProvider(t)
	ts := time.Unix(123, 456)
	linkCtx := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    [16]byte{1},
		SpanID:     [8]byte{2},
		TraceFlags: oteltrace.FlagsSampled,
		Remote:     true,
	})

	ctx, span := StartSpan(
		context.Background(),
		Span().
			TracerName("custom.tracer").
			TracerVersion("1.2.3").
			Kind(oteltrace.SpanKindServer).
			Attributes(attribute.String("a", "b")).
			StartTime(ts).
			Links(oteltrace.Link{SpanContext: linkCtx, Attributes: []attribute.KeyValue{attribute.String("l", "v")}}).
			StartOptions(oteltrace.WithAttributes(attribute.Int("extra", 7))).
			NewRoot(true),
	)
	span.End()
	_ = ctx

	ro := firstSpan(t, exporter)
	if ro.InstrumentationScope.Name != "custom.tracer" {
		t.Fatalf("scope name = %q", ro.InstrumentationScope.Name)
	}
	if ro.InstrumentationScope.Version != "1.2.3" {
		t.Fatalf("scope version = %q", ro.InstrumentationScope.Version)
	}
	if ro.SpanKind != oteltrace.SpanKindServer {
		t.Fatalf("kind = %v", ro.SpanKind)
	}
	if !ro.StartTime.Equal(ts) {
		t.Fatalf("start time = %v, want %v", ro.StartTime, ts)
	}
	if got, ok := spanAttr(ro, "a"); !ok || got.AsString() != "b" {
		t.Fatalf("attr a = %q", got.AsString())
	}
	if got, ok := spanAttr(ro, "extra"); !ok || got.AsInt64() != 7 {
		t.Fatalf("attr extra = %d", got.AsInt64())
	}
	if len(ro.Links) != 1 {
		t.Fatalf("links len = %d", len(ro.Links))
	}
	if ro.Parent.IsValid() {
		t.Fatal("expected new root span to have invalid parent")
	}
}
