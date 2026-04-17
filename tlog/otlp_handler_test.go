package tlog_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	otellog "go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/tigorhutasuhut/telemetry-go/tlog"
)

// captureProcessor is a simple sdklog.Processor that captures emitted records.
type captureProcessor struct {
	records []sdklog.Record
}

func (p *captureProcessor) OnEmit(_ context.Context, r *sdklog.Record) error {
	p.records = append(p.records, r.Clone())
	return nil
}
func (p *captureProcessor) Enabled(_ context.Context, _ sdklog.EnabledParameters) bool { return true }
func (p *captureProcessor) Shutdown(_ context.Context) error                           { return nil }
func (p *captureProcessor) ForceFlush(_ context.Context) error                         { return nil }

func newTestProvider(proc *captureProcessor) otellog.LoggerProvider {
	return sdklog.NewLoggerProvider(sdklog.WithProcessor(proc))
}

func TestOTLPHandler_BasicRecord(t *testing.T) {
	proc := &captureProcessor{}
	provider := newTestProvider(proc)

	h := tlog.NewOTLPHandler(&tlog.OTLPHandlerOptions{
		Provider: provider,
	})
	logger := slog.New(h)

	logger.Info("hello world", "key", "value")

	if len(proc.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(proc.records))
	}
	r := proc.records[0]

	if got := r.Body().AsString(); got != "hello world" {
		t.Errorf("body: got %q, want %q", got, "hello world")
	}
	if r.Severity() != otellog.SeverityInfo {
		t.Errorf("severity: got %v, want Info", r.Severity())
	}
	if r.SeverityText() != "INFO" {
		t.Errorf("severity text: got %q, want INFO", r.SeverityText())
	}

	// Check that key=value is in the attrs.
	found := false
	r.WalkAttributes(func(kv otellog.KeyValue) bool {
		if kv.Key == "key" && kv.Value.AsString() == "value" {
			found = true
			return false
		}
		return true
	})
	if !found {
		t.Error("expected attribute key=value not found")
	}
}

func TestOTLPHandler_SeverityMapping(t *testing.T) {
	cases := []struct {
		level    slog.Level
		wantSev  otellog.Severity
		wantText string
	}{
		{slog.LevelDebug, otellog.SeverityDebug, "DEBUG"},
		{slog.LevelInfo, otellog.SeverityInfo, "INFO"},
		{slog.LevelWarn, otellog.SeverityWarn, "WARN"},
		{slog.LevelError, otellog.SeverityError, "ERROR"},
	}

	for _, tc := range cases {
		t.Run(tc.wantText, func(t *testing.T) {
			proc := &captureProcessor{}
			provider := newTestProvider(proc)
			h := tlog.NewOTLPHandler(&tlog.OTLPHandlerOptions{
				Level:    slog.LevelDebug,
				Provider: provider,
			})
			logger := slog.New(h)
			logger.Log(context.Background(), tc.level, "msg")

			if len(proc.records) != 1 {
				t.Fatalf("expected 1 record, got %d", len(proc.records))
			}
			r := proc.records[0]
			if r.Severity() != tc.wantSev {
				t.Errorf("severity: got %v, want %v", r.Severity(), tc.wantSev)
			}
		})
	}
}

func TestOTLPHandler_TraceCorrelation(t *testing.T) {
	proc := &captureProcessor{}
	provider := newTestProvider(proc)
	h := tlog.NewOTLPHandler(&tlog.OTLPHandlerOptions{Provider: provider})
	logger := slog.New(h)

	traceID := oteltrace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	spanID := oteltrace.SpanID{1, 2, 3, 4, 5, 6, 7, 8}
	sc := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: oteltrace.FlagsSampled,
		Remote:     true,
	})
	ctx := oteltrace.ContextWithSpanContext(context.Background(), sc)

	logger.InfoContext(ctx, "traced")

	if len(proc.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(proc.records))
	}
	r := proc.records[0]

	attrMap := map[string]string{}
	r.WalkAttributes(func(kv otellog.KeyValue) bool {
		if kv.Value.Kind() == otellog.KindString {
			attrMap[kv.Key] = kv.Value.AsString()
		}
		return true
	})

	if attrMap["trace_id"] != traceID.String() {
		t.Errorf("trace_id attr: got %q, want %q", attrMap["trace_id"], traceID.String())
	}
	if attrMap["span_id"] != spanID.String() {
		t.Errorf("span_id attr: got %q, want %q", attrMap["span_id"], spanID.String())
	}
	if attrMap["trace_flags"] == "" {
		t.Error("trace_flags attr missing")
	}
}

func TestOTLPHandler_SourceCodeAttrs(t *testing.T) {
	proc := &captureProcessor{}
	provider := newTestProvider(proc)
	h := tlog.NewOTLPHandler(&tlog.OTLPHandlerOptions{Provider: provider})
	logger := slog.New(h)

	logger.Info("with source")

	if len(proc.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(proc.records))
	}
	r := proc.records[0]

	attrMap := map[string]string{}
	r.WalkAttributes(func(kv otellog.KeyValue) bool {
		if kv.Value.Kind() == otellog.KindString {
			attrMap[kv.Key] = kv.Value.AsString()
		}
		return true
	})

	if attrMap["code.function"] == "" {
		t.Error("code.function missing")
	}
	if attrMap["code.filepath"] == "" {
		t.Error("code.filepath missing")
	}
}

func TestOTLPHandler_WithAttrs(t *testing.T) {
	proc := &captureProcessor{}
	provider := newTestProvider(proc)
	h := tlog.NewOTLPHandler(&tlog.OTLPHandlerOptions{Provider: provider})
	logger := slog.New(h).With("service", "my-svc")

	logger.Info("msg")

	if len(proc.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(proc.records))
	}
	r := proc.records[0]

	found := false
	r.WalkAttributes(func(kv otellog.KeyValue) bool {
		if kv.Key == "service" && kv.Value.AsString() == "my-svc" {
			found = true
			return false
		}
		return true
	})
	if !found {
		t.Error("service attr from WithAttrs not found")
	}
}

func TestOTLPHandler_WithGroup(t *testing.T) {
	proc := &captureProcessor{}
	provider := newTestProvider(proc)
	h := tlog.NewOTLPHandler(&tlog.OTLPHandlerOptions{Provider: provider})
	logger := slog.New(h).WithGroup("http").With("method", "GET")

	logger.Info("request")

	if len(proc.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(proc.records))
	}
	r := proc.records[0]

	found := false
	r.WalkAttributes(func(kv otellog.KeyValue) bool {
		if kv.Key == "http.method" && kv.Value.AsString() == "GET" {
			found = true
			return false
		}
		return true
	})
	if !found {
		t.Error("grouped attr http.method not found")
	}
}

func TestOTLPHandler_ValueNormalization(t *testing.T) {
	proc := &captureProcessor{}
	provider := newTestProvider(proc)
	h := tlog.NewOTLPHandler(&tlog.OTLPHandlerOptions{Provider: provider})
	logger := slog.New(h)

	dur := 5 * time.Second
	raw := json.RawMessage(`{"x":1}`)

	logger.Info("norm",
		"dur", dur,
		"raw", raw,
	)

	if len(proc.records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(proc.records))
	}
	r := proc.records[0]

	attrMap := map[string]string{}
	r.WalkAttributes(func(kv otellog.KeyValue) bool {
		if kv.Value.Kind() == otellog.KindString {
			attrMap[kv.Key] = kv.Value.AsString()
		}
		return true
	})

	if attrMap["dur"] != "5s" {
		t.Errorf("dur: got %q, want %q", attrMap["dur"], "5s")
	}
	if attrMap["raw"] != `{"x":1}` {
		t.Errorf("raw: got %q, want %q", attrMap["raw"], `{"x":1}`)
	}
}

func TestOTLPHandler_Enabled_MinLevel(t *testing.T) {
	proc := &captureProcessor{}
	provider := newTestProvider(proc)
	h := tlog.NewOTLPHandler(&tlog.OTLPHandlerOptions{
		Level:    slog.LevelWarn,
		Provider: provider,
	})

	if h.Enabled(context.Background(), slog.LevelInfo) {
		t.Error("expected Info to be disabled when min level is Warn")
	}
	if !h.Enabled(context.Background(), slog.LevelWarn) {
		t.Error("expected Warn to be enabled")
	}
	if !h.Enabled(context.Background(), slog.LevelError) {
		t.Error("expected Error to be enabled")
	}
}

func TestOTLPHandler_DefaultProvider(t *testing.T) {
	// Constructing with nil provider must not panic; it uses the global provider.
	h := tlog.NewOTLPHandler(nil)
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}
