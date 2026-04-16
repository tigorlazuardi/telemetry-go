package tlog

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/tigorhutasuhut/telemetry-go/tcaller"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func captureDefaultLogger(t *testing.T) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	old := slog.Default()
	slog.SetDefault(NewLogger(buf, &StderrHandlerOptions{IsTTY: boolPtr(false)}))
	t.Cleanup(func() { slog.SetDefault(old) })
	return buf
}

func boolPtr(v bool) *bool { return &v }

func decodeRecord(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &m); err != nil {
		t.Fatalf("json.Unmarshal error = %v, body=%q", err, buf.String())
	}
	return m
}

func emitLog() {
	Log(context.Background(), Opt().Message("hello"))
}

func emitEntry() {
	New().Message("hello").Print(context.Background())
}

//go:noinline
func currentTestCaller() tcaller.Caller {
	pc, _, _, _ := runtime.Caller(0)
	return tcaller.FromPC(pc)
}

func TestLogSkipsEmptyEntry(t *testing.T) {
	buf := captureDefaultLogger(t)
	Log(context.Background())
	if buf.Len() != 0 {
		t.Fatalf("expected no output, got %q", buf.String())
	}
}

func TestLogDefaultsAndFields(t *testing.T) {
	buf := captureDefaultLogger(t)
	emitLog()
	rec := decodeRecord(t, buf)

	if rec["level"] != "INFO" {
		t.Fatalf("level = %v", rec["level"])
	}
	if rec["msg"] != "hello" {
		t.Fatalf("msg = %v", rec["msg"])
	}
	source, ok := rec["source"].(map[string]any)
	if !ok {
		t.Fatalf("source = %#v", rec["source"])
	}
	if source["file"] != "/Users/tigorhutasuhut/Projects/telemetry-go/tlog/log_test.go" {
		t.Fatalf("source.file = %v", source["file"])
	}
	if source["function"] != "github.com/tigorhutasuhut/telemetry-go/tlog.emitLog" {
		t.Fatalf("source.function = %v", source["function"])
	}
	if _, ok := source["line"].(float64); !ok {
		t.Fatalf("source.line = %#v", source["line"])
	}
}

func TestEntryPrintDefaultsCallerToPrintCallsite(t *testing.T) {
	buf := captureDefaultLogger(t)
	emitEntry()
	rec := decodeRecord(t, buf)
	source := rec["source"].(map[string]any)
	if source["function"] != "github.com/tigorhutasuhut/telemetry-go/tlog.emitEntry" {
		t.Fatalf("source.function = %v", source["function"])
	}
}

func TestLogUsesContextAttrsAndCaller(t *testing.T) {
	buf := captureDefaultLogger(t)
	ctx := WithAttrs(context.Background(), slog.String("request_id", "r1"))
	ctx = WithCaller(ctx, currentTestCaller())
	Log(ctx, Opt().Message("hello").Fields("user", "u1"))
	rec := decodeRecord(t, buf)

	if rec["request_id"] != "r1" {
		t.Fatalf("request_id = %v", rec["request_id"])
	}
	if rec["user"] != "u1" {
		t.Fatalf("user = %v", rec["user"])
	}
	source := rec["source"].(map[string]any)
	if source["function"] != "github.com/tigorhutasuhut/telemetry-go/tlog.currentTestCaller" {
		t.Fatalf("source.function = %v", source["function"])
	}
}

func TestWithAttrsImmutableAndSiblingIsolation(t *testing.T) {
	base := WithAttrs(context.Background(), slog.String("a", "1"))
	left := WithAttrs(base, slog.String("b", "2"))
	right := WithAttrs(base, slog.String("c", "3"))

	baseAttrs := AttrsFromContext(base)
	leftAttrs := AttrsFromContext(left)
	rightAttrs := AttrsFromContext(right)

	if len(baseAttrs) != 1 || baseAttrs[0].Key != "a" {
		t.Fatalf("base attrs = %#v", baseAttrs)
	}
	if len(leftAttrs) != 2 || leftAttrs[1].Key != "b" {
		t.Fatalf("left attrs = %#v", leftAttrs)
	}
	if len(rightAttrs) != 2 || rightAttrs[1].Key != "c" {
		t.Fatalf("right attrs = %#v", rightAttrs)
	}
}

func TestWithCurrentCaller(t *testing.T) {
	ctx := WithCurrentCaller(context.Background())
	caller, ok := CallerFromContext(ctx)
	if !ok || caller.IsZero() {
		t.Fatal("expected caller in context")
	}
	if caller.ShortFunction() != "tlog.TestWithCurrentCaller" {
		t.Fatalf("caller.ShortFunction() = %q", caller.ShortFunction())
	}
}

func TestTraceCorrelationFields(t *testing.T) {
	buf := captureDefaultLogger(t)
	sc := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    [16]byte{1, 2, 3},
		SpanID:     [8]byte{4, 5, 6},
		TraceFlags: oteltrace.FlagsSampled,
	})
	ctx := oteltrace.ContextWithSpanContext(context.Background(), sc)
	Log(ctx, Opt().Message("hello"))
	rec := decodeRecord(t, buf)

	if rec["trace_id"] != sc.TraceID().String() {
		t.Fatalf("trace_id = %v", rec["trace_id"])
	}
	if rec["span_id"] != sc.SpanID().String() {
		t.Fatalf("span_id = %v", rec["span_id"])
	}
	if rec["trace_flags"] != sc.TraceFlags().String() {
		t.Fatalf("trace_flags = %v", rec["trace_flags"])
	}
}

func TestClassicCompatibility(t *testing.T) {
	buf := captureDefaultLogger(t)
	Print("hello", " ", "world")
	rec := decodeRecord(t, buf)
	if rec["msg"] != "hello world" {
		t.Fatalf("msg = %v", rec["msg"])
	}

	buf.Reset()
	Printf("user=%s", "u1")
	rec = decodeRecord(t, buf)
	if rec["msg"] != "user=u1" {
		t.Fatalf("msg = %v", rec["msg"])
	}

	buf.Reset()
	Println("a", "b")
	rec = decodeRecord(t, buf)
	if rec["msg"] != "a b" {
		t.Fatalf("msg = %v", rec["msg"])
	}
}

func TestEntryTimeAndLevel(t *testing.T) {
	buf := captureDefaultLogger(t)
	ts := time.Unix(100, 0)
	New().Level(slog.LevelWarn).Time(ts).Message("hello").Print(context.Background())
	rec := decodeRecord(t, buf)
	if rec["level"] != "WARN" {
		t.Fatalf("level = %v", rec["level"])
	}
}

func TestEntryUsesExplicitLogger(t *testing.T) {
	defaultBuf := captureDefaultLogger(t)
	explicitBuf := &bytes.Buffer{}
	logger := NewLogger(explicitBuf, &StderrHandlerOptions{IsTTY: boolPtr(false)})

	New().Logger(logger).Message("hello").Print(context.Background())

	if defaultBuf.Len() != 0 {
		t.Fatalf("default logger should not be used, got %q", defaultBuf.String())
	}
	rec := decodeRecord(t, explicitBuf)
	if rec["msg"] != "hello" {
		t.Fatalf("msg = %v", rec["msg"])
	}
}

func TestClassicLoggerUsesExplicitLogger(t *testing.T) {
	defaultBuf := captureDefaultLogger(t)
	explicitBuf := &bytes.Buffer{}
	logger := NewLogger(explicitBuf, &StderrHandlerOptions{IsTTY: boolPtr(false)})

	Classic().Logger(logger).Printf("user=%s", "u1")

	if defaultBuf.Len() != 0 {
		t.Fatalf("default logger should not be used, got %q", defaultBuf.String())
	}
	rec := decodeRecord(t, explicitBuf)
	if rec["msg"] != "user=u1" {
		t.Fatalf("msg = %v", rec["msg"])
	}
}

func TestClassicLoggerLevels(t *testing.T) {
	buf := captureDefaultLogger(t)
	logger := Classic()

	logger.Debug("debug")
	rec := decodeRecord(t, buf)
	if rec["level"] != "DEBUG" || rec["msg"] != "debug" {
		t.Fatalf("debug record = %#v", rec)
	}

	buf.Reset()
	logger.Infof("hello %s", "info")
	rec = decodeRecord(t, buf)
	if rec["level"] != "INFO" || rec["msg"] != "hello info" {
		t.Fatalf("info record = %#v", rec)
	}

	buf.Reset()
	logger.Warn("warn")
	rec = decodeRecord(t, buf)
	if rec["level"] != "WARN" || rec["msg"] != "warn" {
		t.Fatalf("warn record = %#v", rec)
	}

	buf.Reset()
	logger.Errorf("err=%d", 7)
	rec = decodeRecord(t, buf)
	if rec["level"] != "ERROR" || rec["msg"] != "err=7" {
		t.Fatalf("error record = %#v", rec)
	}
}

func TestTTYHandlerFormatting(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, &StderrHandlerOptions{IsTTY: boolPtr(true)})
	New().Logger(logger).Message("hello").Fields("user", "u1").Print(context.Background())
	out := buf.String()
	if !strings.Contains(out, "hello") {
		t.Fatalf("output = %q", out)
	}
	if !strings.Contains(out, "\x1b[") {
		t.Fatalf("expected colored output, got %q", out)
	}
	if !strings.Contains(out, "hello\n") {
		t.Fatalf("expected details on next line, got %q", out)
	}
	if !strings.Contains(out, `"user"`) || !strings.Contains(out, `"u1"`) {
		t.Fatalf("expected pretty details, got %q", out)
	}
	if !strings.Contains(out, "<log_test.go:") {
		t.Fatalf("expected caller in header, got %q", out)
	}
	if !strings.HasSuffix(out, "\n\n") {
		t.Fatalf("expected trailing blank line, got %q", out)
	}
}

func TestTTYHandlerOmitsEmptyDetailsBlock(t *testing.T) {
	buf := &bytes.Buffer{}
	logger := NewLogger(buf, &StderrHandlerOptions{IsTTY: boolPtr(true)})
	New().Logger(logger).Message("hello").Print(context.Background())
	out := buf.String()
	if strings.Contains(out, "{}") {
		t.Fatalf("expected no empty details object, got %q", out)
	}
	if !strings.HasSuffix(out, "\n\n") {
		t.Fatalf("expected trailing blank line, got %q", out)
	}
}
