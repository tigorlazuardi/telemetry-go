package terror_test

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/tigorhutasuhut/telemetry-go/tcaller"
	"github.com/tigorhutasuhut/telemetry-go/terror"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// asError asserts that err contains a terror.Error in its chain and returns it.
func asError(t *testing.T, err error) terror.Error {
	t.Helper()
	var e terror.Error
	if !errors.As(err, &e) {
		t.Fatalf("expected terror.Error in chain, got %T: %v", err, err)
	}
	return e
}

// ---------------------------------------------------------------------------
// Code / StatusCode
// ---------------------------------------------------------------------------

func TestStatusCode_HTTPStatusCode(t *testing.T) {
	cases := []struct {
		code terror.StatusCode
		want int
	}{
		{terror.CodeBadRequest, http.StatusBadRequest},
		{terror.CodeUnauthorized, http.StatusUnauthorized},
		{terror.CodeForbidden, http.StatusForbidden},
		{terror.CodeNotFound, http.StatusNotFound},
		{terror.CodeConflict, http.StatusConflict},
		{terror.CodeUnprocessableEntity, http.StatusUnprocessableEntity},
		{terror.CodeTooManyRequests, http.StatusTooManyRequests},
		{terror.CodeInternal, http.StatusInternalServerError},
		{terror.CodeServiceUnavailable, http.StatusServiceUnavailable},
		{terror.CodeGatewayTimeout, http.StatusGatewayTimeout},
	}
	for _, tc := range cases {
		t.Run(tc.code.String(), func(t *testing.T) {
			if got := tc.code.HTTPStatusCode(); got != tc.want {
				t.Errorf("HTTPStatusCode() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestStatusCode_String_KnownStatus(t *testing.T) {
	if got := terror.CodeNotFound.String(); got != "Not Found" {
		t.Errorf("String() = %q, want %q", got, "Not Found")
	}
}

func TestStatusCode_String_UnknownStatus(t *testing.T) {
	code := terror.StatusCode(999)
	if got := code.String(); got != "StatusCode(999)" {
		t.Errorf("String() = %q, want %q", got, "StatusCode(999)")
	}
}

func TestStatusCode_ImplementsCode(t *testing.T) {
	var _ terror.Code = terror.CodeInternal
}

// ---------------------------------------------------------------------------
// Error interface
// ---------------------------------------------------------------------------

func TestError_Error_Message(t *testing.T) {
	err := terror.Fail(terror.Err().Message("something went wrong"))
	if got := err.Error(); got != "something went wrong" {
		t.Errorf("Error() = %q, want %q", got, "something went wrong")
	}
}

func TestError_Error_CodeFallback(t *testing.T) {
	err := terror.Fail(terror.Err().Code(terror.CodeNotFound))
	if got := err.Error(); got != "Not Found" {
		t.Errorf("Error() = %q, want %q", got, "Not Found")
	}
}

func TestError_Error_MessageTakesPriorityOverCode(t *testing.T) {
	err := terror.Fail(terror.Err().Message("custom").Code(terror.CodeInternal))
	if got := err.Error(); got != "custom" {
		t.Errorf("Error() = %q, want %q", got, "custom")
	}
}

func TestError_Error_Empty(t *testing.T) {
	err := terror.Fail()
	if got := err.Error(); got != "" {
		t.Errorf("Error() = %q, want empty", got)
	}
}

func TestError_Unwrap_SingleCause(t *testing.T) {
	cause := errors.New("root cause")
	wrapped := terror.Wrap(cause, terror.Err().Message("wrapper"))
	if !errors.Is(wrapped, cause) {
		t.Error("errors.Is: cause not found in chain")
	}
}

func TestError_Unwrap_MultipleCauses(t *testing.T) {
	cause1 := errors.New("cause1")
	cause2 := errors.New("cause2")

	err := terror.Wrap(cause1, terror.Err().Message("multi"))
	e := asError(t, err)
	e.Join(cause2)

	if !errors.Is(err, cause1) {
		t.Error("errors.Is: cause1 not found")
	}
	if !errors.Is(err, cause2) {
		t.Error("errors.Is: cause2 not found")
	}
}

func TestError_Unwrap_NilCause(t *testing.T) {
	err := terror.Fail(terror.Err().Message("no cause"))
	e := asError(t, err)
	if e.Unwrap() != nil {
		t.Error("Unwrap() should be nil when no Source")
	}
}

func TestError_ImplementsError(t *testing.T) {
	var _ error = terror.Fail()
}

// ---------------------------------------------------------------------------
// Wrap / Fail
// ---------------------------------------------------------------------------

func TestWrap_Message(t *testing.T) {
	cause := errors.New("inner")
	err := terror.Wrap(cause, terror.Err().Message("outer"))
	e := asError(t, err)
	if e.Message() != "outer" {
		t.Errorf("Message() = %q, want %q", e.Message(), "outer")
	}
	if !errors.Is(err, cause) {
		t.Error("cause not in chain")
	}
}

func TestWrap_Code(t *testing.T) {
	cause := errors.New("db error")
	err := terror.Wrap(cause, terror.Err().Code(terror.CodeInternal))
	e := asError(t, err)
	if e.Code() != terror.CodeInternal {
		t.Errorf("Code() = %v, want CodeInternal", e.Code())
	}
}

func TestWrap_Fields(t *testing.T) {
	cause := errors.New("cause")
	err := terror.Wrap(cause, terror.Err().Fields("key", "val"))
	e := asError(t, err)
	if len(e.Fields()) != 2 {
		t.Fatalf("Fields() length = %d, want 2", len(e.Fields()))
	}
}

func TestWrap_CallerCaptured(t *testing.T) {
	cause := errors.New("cause")
	err := terror.Wrap(cause)
	e := asError(t, err)
	if e.Caller().IsZero() {
		t.Error("Caller should be captured automatically")
	}
	if file := e.Caller().File(); !strings.Contains(file, "terror_test") {
		t.Errorf("Caller().File() = %q, want to contain 'terror_test'", file)
	}
}

func TestWrap_CallerOverride(t *testing.T) {
	cause := errors.New("cause")
	from := tcaller.New(0)
	err := terror.Wrap(cause, terror.Err().Caller(from))
	e := asError(t, err)
	if e.Caller() != from {
		t.Error("explicit Caller not honored")
	}
}

func TestFail_Message(t *testing.T) {
	err := terror.Fail(terror.Err().Message("standalone"))
	e := asError(t, err)
	if e.Message() != "standalone" {
		t.Errorf("Message() = %q, want %q", e.Message(), "standalone")
	}
	if e.Unwrap() != nil {
		t.Error("Fail should have no Source")
	}
}

func TestFail_Code(t *testing.T) {
	err := terror.Fail(terror.Err().Code(terror.CodeBadRequest))
	e := asError(t, err)
	if e.Code() != terror.CodeBadRequest {
		t.Errorf("Code() = %v, want CodeBadRequest", e.Code())
	}
}

func TestFail_CallerCaptured(t *testing.T) {
	err := terror.Fail()
	e := asError(t, err)
	if e.Caller().IsZero() {
		t.Error("Caller should be captured automatically")
	}
	if file := e.Caller().File(); !strings.Contains(file, "terror_test") {
		t.Errorf("Caller().File() = %q, want to contain 'terror_test'", file)
	}
}

// TestFail_MultipleOptions verifies last-write-wins for singular fields and
// append semantics for slice fields.
func TestFail_MultipleOptions(t *testing.T) {
	err := terror.Fail(
		terror.Err().Message("first").Code(terror.CodeBadRequest).Fields("a", 1),
		terror.Err().Message("second").Fields("b", 2),
	)
	e := asError(t, err)
	if e.Message() != "second" {
		t.Errorf("Message() = %q, want %q", e.Message(), "second")
	}
	if e.Code() != terror.CodeBadRequest {
		t.Errorf("Code() = %v, want CodeBadRequest", e.Code())
	}
	if len(e.Fields()) != 4 {
		t.Errorf("Fields() length = %d, want 4", len(e.Fields()))
	}
}

// ---------------------------------------------------------------------------
// Wrapf
// ---------------------------------------------------------------------------

func TestWrapf_FormatsMessage(t *testing.T) {
	cause := errors.New("root")
	err := terror.Wrapf(cause, "request failed: %s (code=%d)", "timeout", 503)
	e := asError(t, err)
	want := "request failed: timeout (code=503)"
	if e.Message() != want {
		t.Errorf("Message() = %q, want %q", e.Message(), want)
	}
}

func TestWrapf_WrapsCause(t *testing.T) {
	cause := errors.New("underlying")
	err := terror.Wrapf(cause, "wrapped")
	if !errors.Is(err, cause) {
		t.Error("cause not in chain")
	}
}

func TestWrapf_NoFields(t *testing.T) {
	err := terror.Wrapf(errors.New("x"), "msg")
	e := asError(t, err)
	if len(e.Fields()) != 0 {
		t.Errorf("Fields() should be empty, got %v", e.Fields())
	}
}

func TestWrapf_CallerCaptured(t *testing.T) {
	cause := errors.New("c")
	err := terror.Wrapf(cause, "msg")
	e := asError(t, err)
	if e.Caller().IsZero() {
		t.Error("Caller should be captured")
	}
	if file := e.Caller().File(); !strings.Contains(file, "terror_test") {
		t.Errorf("Caller().File() = %q, want to contain 'terror_test'", file)
	}
}

func TestWrapf_NoExtraArg(t *testing.T) {
	err := terror.Wrapf(errors.New("x"), "literal message")
	e := asError(t, err)
	if e.Message() != "literal message" {
		t.Errorf("Message() = %q, want %q", e.Message(), "literal message")
	}
}

// ---------------------------------------------------------------------------
// Wrapw
// ---------------------------------------------------------------------------

func TestWrapw_Message(t *testing.T) {
	cause := errors.New("cause")
	err := terror.Wrapw(cause, "wrapped with fields", "user", "alice", "attempt", 3)
	e := asError(t, err)
	if e.Message() != "wrapped with fields" {
		t.Errorf("Message() = %q, want %q", e.Message(), "wrapped with fields")
	}
}

func TestWrapw_Fields(t *testing.T) {
	cause := errors.New("cause")
	err := terror.Wrapw(cause, "msg", "k1", "v1", "k2", 42)
	e := asError(t, err)
	if len(e.Fields()) != 4 {
		t.Fatalf("Fields() length = %d, want 4", len(e.Fields()))
	}
	if e.Fields()[0] != "k1" {
		t.Errorf("Fields()[0] = %v, want %q", e.Fields()[0], "k1")
	}
	if e.Fields()[2] != "k2" {
		t.Errorf("Fields()[2] = %v, want %q", e.Fields()[2], "k2")
	}
}

func TestWrapw_WrapsCause(t *testing.T) {
	cause := errors.New("db")
	err := terror.Wrapw(cause, "service error")
	if !errors.Is(err, cause) {
		t.Error("cause not in chain")
	}
}

func TestWrapw_NoFields(t *testing.T) {
	err := terror.Wrapw(errors.New("x"), "msg")
	e := asError(t, err)
	if len(e.Fields()) != 0 {
		t.Errorf("Fields() should be empty, got %v", e.Fields())
	}
}

func TestWrapw_CallerCaptured(t *testing.T) {
	err := terror.Wrapw(errors.New("x"), "msg")
	e := asError(t, err)
	if e.Caller().IsZero() {
		t.Error("Caller should be captured")
	}
	if file := e.Caller().File(); !strings.Contains(file, "terror_test") {
		t.Errorf("Caller().File() = %q, want to contain 'terror_test'", file)
	}
}

func TestWrapw_SlogAttrField(t *testing.T) {
	cause := errors.New("x")
	attr := slog.String("component", "auth")
	err := terror.Wrapw(cause, "msg", attr)
	e := asError(t, err)
	if len(e.Fields()) != 1 {
		t.Fatalf("Fields() length = %d, want 1", len(e.Fields()))
	}
	got, ok := e.Fields()[0].(slog.Attr)
	if !ok || !got.Equal(attr) {
		t.Errorf("Fields()[0] = %v, want %v", e.Fields()[0], attr)
	}
}

// ---------------------------------------------------------------------------
// Failf
// ---------------------------------------------------------------------------

func TestFailf_FormatsMessage(t *testing.T) {
	err := terror.Failf("invalid input: field=%s value=%d", "age", -1)
	e := asError(t, err)
	want := "invalid input: field=age value=-1"
	if e.Message() != want {
		t.Errorf("Message() = %q, want %q", e.Message(), want)
	}
}

func TestFailf_NoSource(t *testing.T) {
	err := terror.Failf("standalone %s", "error")
	e := asError(t, err)
	if e.Unwrap() != nil {
		t.Error("Failf should produce no Source")
	}
}

func TestFailf_NoFields(t *testing.T) {
	err := terror.Failf("msg")
	e := asError(t, err)
	if len(e.Fields()) != 0 {
		t.Errorf("Fields() should be empty, got %v", e.Fields())
	}
}

func TestFailf_CallerCaptured(t *testing.T) {
	err := terror.Failf("msg")
	e := asError(t, err)
	if e.Caller().IsZero() {
		t.Error("Caller should be captured")
	}
	if file := e.Caller().File(); !strings.Contains(file, "terror_test") {
		t.Errorf("Caller().File() = %q, want to contain 'terror_test'", file)
	}
}

func TestFailf_NoExtraArg(t *testing.T) {
	err := terror.Failf("no args here")
	e := asError(t, err)
	if e.Message() != "no args here" {
		t.Errorf("Message() = %q, want %q", e.Message(), "no args here")
	}
}

// ---------------------------------------------------------------------------
// Failw
// ---------------------------------------------------------------------------

func TestFailw_Message(t *testing.T) {
	err := terror.Failw("validation failed", "field", "email", "reason", "invalid format")
	e := asError(t, err)
	if e.Message() != "validation failed" {
		t.Errorf("Message() = %q, want %q", e.Message(), "validation failed")
	}
}

func TestFailw_Fields(t *testing.T) {
	err := terror.Failw("msg", "x", 1, "y", 2)
	e := asError(t, err)
	if len(e.Fields()) != 4 {
		t.Fatalf("Fields() length = %d, want 4", len(e.Fields()))
	}
	if e.Fields()[0] != "x" {
		t.Errorf("Fields()[0] = %v, want %q", e.Fields()[0], "x")
	}
}

func TestFailw_NoSource(t *testing.T) {
	err := terror.Failw("msg")
	e := asError(t, err)
	if e.Unwrap() != nil {
		t.Error("Failw should produce no Source")
	}
}

func TestFailw_NoFields(t *testing.T) {
	err := terror.Failw("msg")
	e := asError(t, err)
	if len(e.Fields()) != 0 {
		t.Errorf("Fields() should be empty, got %v", e.Fields())
	}
}

func TestFailw_CallerCaptured(t *testing.T) {
	err := terror.Failw("msg")
	e := asError(t, err)
	if e.Caller().IsZero() {
		t.Error("Caller should be captured")
	}
	if file := e.Caller().File(); !strings.Contains(file, "terror_test") {
		t.Errorf("Caller().File() = %q, want to contain 'terror_test'", file)
	}
}

func TestFailw_SlogAttrField(t *testing.T) {
	attr := slog.Int("retry", 3)
	err := terror.Failw("msg", attr)
	e := asError(t, err)
	if len(e.Fields()) != 1 {
		t.Fatalf("Fields() length = %d, want 1", len(e.Fields()))
	}
	got, ok := e.Fields()[0].(slog.Attr)
	if !ok || !got.Equal(attr) {
		t.Errorf("Fields()[0] = %v, want %v", e.Fields()[0], attr)
	}
}

// ---------------------------------------------------------------------------
// ErrorOptions builder
// ---------------------------------------------------------------------------

func TestErrOptions_NilSafe(t *testing.T) {
	// Passing a nil option must not panic.
	err := terror.Fail(nil)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
}

// ---------------------------------------------------------------------------
// LogValue
// ---------------------------------------------------------------------------

func TestError_LogValue_Message(t *testing.T) {
	err := terror.Fail(terror.Err().Message("log this"))
	e := asError(t, err)
	lv := e.LogValue()
	found := false
	for _, attr := range lv.Group() {
		if attr.Key == "message" && attr.Value.String() == "log this" {
			found = true
		}
	}
	if !found {
		t.Errorf("LogValue group missing message=log this; got %v", lv)
	}
}

func TestError_LogValue_NoMessageWhenEmpty(t *testing.T) {
	err := terror.Fail(terror.Err().Code(terror.CodeInternal))
	e := asError(t, err)
	lv := e.LogValue()
	for _, attr := range lv.Group() {
		if attr.Key == "message" {
			t.Error("LogValue should omit message key when message is empty")
		}
	}
}

func TestError_LogValue_Code(t *testing.T) {
	err := terror.Fail(terror.Err().Code(terror.CodeNotFound))
	e := asError(t, err)
	lv := e.LogValue()
	found := false
	for _, attr := range lv.Group() {
		if attr.Key == "code" {
			found = true
		}
	}
	if !found {
		t.Error("LogValue group missing code key")
	}
}

func TestError_LogValue_CallerPresent(t *testing.T) {
	err := terror.Fail()
	e := asError(t, err)
	lv := e.LogValue()
	found := false
	for _, attr := range lv.Group() {
		if attr.Key == "caller" {
			found = true
		}
	}
	if !found {
		t.Error("LogValue group missing caller key")
	}
}

func TestError_LogValue_SingleSource(t *testing.T) {
	cause := errors.New("root")
	err := terror.Wrap(cause, terror.Err().Message("wrap"))
	e := asError(t, err)
	lv := e.LogValue()
	found := false
	for _, attr := range lv.Group() {
		if attr.Key == "source" {
			found = true
		}
	}
	if !found {
		t.Error("LogValue group missing source key for wrapped error")
	}
}

func TestError_LogValue_NoSourceWhenNil(t *testing.T) {
	err := terror.Fail(terror.Err().Message("standalone"))
	e := asError(t, err)
	lv := e.LogValue()
	for _, attr := range lv.Group() {
		if attr.Key == "source" {
			t.Error("LogValue should omit source when no cause")
		}
	}
}

func TestError_LogValue_MultipleSource(t *testing.T) {
	cause1 := errors.New("c1")
	cause2 := errors.New("c2")
	err := terror.Wrap(cause1, terror.Err().Message("multi"))
	e := asError(t, err)
	e.Join(cause2)

	lv := e.LogValue()
	found := false
	for _, attr := range lv.Group() {
		if attr.Key == "source" {
			found = true
			if attr.Value.Kind() != slog.KindGroup {
				t.Error("source should be a group for multiple causes")
			}
		}
	}
	if !found {
		t.Error("LogValue group missing source key")
	}
}

func TestStatusCode_LogValue(t *testing.T) {
	lv := terror.CodeInternal.LogValue()
	if lv.Kind() != slog.KindString {
		t.Errorf("LogValue kind = %v, want String", lv.Kind())
	}
	if got := lv.String(); got != "Internal Server Error" {
		t.Errorf("LogValue = %q, want %q", got, "Internal Server Error")
	}
}

// ---------------------------------------------------------------------------
// errors.As integration
// ---------------------------------------------------------------------------

func TestErrorsAs_ThroughWrapChain(t *testing.T) {
	cause := terror.Fail(terror.Err().Message("root").Code(terror.CodeNotFound))
	wrapped := terror.Wrap(cause, terror.Err().Message("wrapper"))

	var target terror.Error
	if !errors.As(wrapped, &target) {
		t.Fatal("errors.As should find terror.Error in chain")
	}
	// The outermost error is 'wrapped'.
	if target.Message() != "wrapper" {
		t.Errorf("Message() = %q, want %q", target.Message(), "wrapper")
	}
}

func TestErrorsIs_PlainErrorInChain(t *testing.T) {
	sentinel := fmt.Errorf("sentinel")
	err := terror.Wrapf(sentinel, "context: %s", "db")
	if !errors.Is(err, sentinel) {
		t.Error("errors.Is should find sentinel through Wrapf")
	}
}

// ---------------------------------------------------------------------------
// Caller depth correctness
// ---------------------------------------------------------------------------

func TestCallerDepth_Wrap(t *testing.T) {
	err := terror.Wrap(errors.New("x")) //nolint:goerr113
	e := asError(t, err)
	checkCallerLine(t, e, "TestCallerDepth_Wrap")
}

func TestCallerDepth_Fail(t *testing.T) {
	err := terror.Fail()
	e := asError(t, err)
	checkCallerLine(t, e, "TestCallerDepth_Fail")
}

func TestCallerDepth_Wrapf(t *testing.T) {
	err := terror.Wrapf(errors.New("x"), "msg")
	e := asError(t, err)
	checkCallerLine(t, e, "TestCallerDepth_Wrapf")
}

func TestCallerDepth_Wrapw(t *testing.T) {
	err := terror.Wrapw(errors.New("x"), "msg")
	e := asError(t, err)
	checkCallerLine(t, e, "TestCallerDepth_Wrapw")
}

func TestCallerDepth_Failf(t *testing.T) {
	err := terror.Failf("msg")
	e := asError(t, err)
	checkCallerLine(t, e, "TestCallerDepth_Failf")
}

func TestCallerDepth_Failw(t *testing.T) {
	err := terror.Failw("msg")
	e := asError(t, err)
	checkCallerLine(t, e, "TestCallerDepth_Failw")
}

func TestCallerDepth_Multi(t *testing.T) {
	e := terror.Multi(errors.New("x"))
	checkCallerLine(t, e, "TestCallerDepth_Multi")
}

func checkCallerLine(t *testing.T, e terror.Error, wantFunc string) {
	t.Helper()
	fn := e.Caller().Function()
	if !strings.Contains(fn, wantFunc) {
		t.Errorf("Caller().Function() = %q, want to contain %q", fn, wantFunc)
	}
}

// ---------------------------------------------------------------------------
// Multi (package-level constructor)
// ---------------------------------------------------------------------------

func TestMulti_DiscardsNil(t *testing.T) {
	e := terror.Multi(nil, errors.New("a"), nil, errors.New("b"), nil)
	if len(e.Source()) != 2 {
		t.Fatalf("Source() length = %d, want 2", len(e.Source()))
	}
}

func TestMulti_AllNil(t *testing.T) {
	e := terror.Multi(nil, nil)
	if len(e.Source()) != 0 {
		t.Errorf("Source() should be empty when all inputs are nil, got %d", len(e.Source()))
	}
}

func TestMulti_NoArgs(t *testing.T) {
	e := terror.Multi()
	if e == nil {
		t.Fatal("Multi() must return non-nil Error")
	}
	if len(e.Source()) != 0 {
		t.Errorf("Source() should be empty, got %d", len(e.Source()))
	}
}

func TestMulti_ErrorsIsEach(t *testing.T) {
	a := errors.New("a")
	b := errors.New("b")
	e := terror.Multi(a, b)
	if !errors.Is(e, a) {
		t.Error("errors.Is: a not found")
	}
	if !errors.Is(e, b) {
		t.Error("errors.Is: b not found")
	}
}

func TestMulti_CallerCaptured(t *testing.T) {
	e := terror.Multi(errors.New("x"))
	if e.Caller().IsZero() {
		t.Error("Caller should be captured")
	}
	if file := e.Caller().File(); !strings.Contains(file, "terror_test") {
		t.Errorf("Caller().File() = %q, want to contain 'terror_test'", file)
	}
}

func TestMulti_SetMessageChainable(t *testing.T) {
	// Multi returns Error interface — SetMessage mutates in place and
	// returns the same interface for chaining.
	e := terror.Multi(errors.New("x"))
	e.SetMessage("combined failure")
	if e.Error() != "combined failure" {
		t.Errorf("Error() = %q, want %q", e.Error(), "combined failure")
	}
}

// ---------------------------------------------------------------------------
// (Error).Join (method)
// ---------------------------------------------------------------------------

func TestErrorJoin_AppendsNonNil(t *testing.T) {
	e := terror.Multi(errors.New("a"))
	e.Join(errors.New("b"), errors.New("c"))
	if len(e.Source()) != 3 {
		t.Fatalf("Source() length = %d, want 3", len(e.Source()))
	}
}

func TestErrorJoin_DiscardsNil(t *testing.T) {
	e := terror.Multi(errors.New("a"))
	e.Join(nil, errors.New("b"), nil)
	if len(e.Source()) != 2 {
		t.Fatalf("Source() length = %d, want 2", len(e.Source()))
	}
}

func TestErrorJoin_AllNil(t *testing.T) {
	e := terror.Multi()
	e.Join(nil, nil)
	if len(e.Source()) != 0 {
		t.Errorf("Source() should remain empty, got %d", len(e.Source()))
	}
}

func TestErrorJoin_Chainable(t *testing.T) {
	a, b := errors.New("a"), errors.New("b")
	e := terror.Multi().Join(a).Join(b)
	if len(e.Source()) != 2 {
		t.Fatalf("Source() length = %d, want 2", len(e.Source()))
	}
}

// ---------------------------------------------------------------------------
// (Error).Resolve
// ---------------------------------------------------------------------------

func TestResolve_ReturnsNilWhenSourceEmpty(t *testing.T) {
	e := terror.Multi()
	if err := e.Resolve(); err != nil {
		t.Errorf("Resolve() = %v, want nil", err)
	}
}

func TestResolve_ReturnsNilWhenAllSourceNil(t *testing.T) {
	e := terror.Multi()
	e.SetSource([]error{nil, nil})
	if err := e.Resolve(); err != nil {
		t.Errorf("Resolve() = %v, want nil", err)
	}
}

func TestResolve_ReturnsErrorWhenSourceNonEmpty(t *testing.T) {
	e := terror.Multi(errors.New("x"))
	if err := e.Resolve(); err == nil {
		t.Error("Resolve() = nil, want non-nil error")
	}
}

func TestResolve_CleansNilsFromSource(t *testing.T) {
	e := terror.Multi()
	e.SetSource([]error{nil, errors.New("a"), nil, errors.New("b"), nil})
	err := e.Resolve()
	if err == nil {
		t.Fatal("Resolve() = nil, want non-nil")
	}
	te := asError(t, err)
	if len(te.Source()) != 2 {
		t.Errorf("Source() after Resolve = %d, want 2", len(te.Source()))
	}
}

func TestResolve_IsErrorInterface(t *testing.T) {
	e := terror.Multi(errors.New("x"))
	err := e.Resolve()
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	var te terror.Error
	if !errors.As(err, &te) {
		t.Error("Resolve() result should unwrap to terror.Error")
	}
}

func TestResolve_MultiJoinPattern(t *testing.T) {
	mayFail := func(fail bool) error {
		if fail {
			return errors.New("failed")
		}
		return nil
	}

	e := terror.Multi().Join(mayFail(false), mayFail(false))
	if err := e.Resolve(); err != nil {
		t.Errorf("all-pass: Resolve() = %v, want nil", err)
	}

	e2 := terror.Multi().Join(mayFail(true), mayFail(false), mayFail(true))
	err := e2.Resolve()
	if err == nil {
		t.Fatal("some-fail: Resolve() = nil, want non-nil")
	}
	te := asError(t, err)
	if len(te.Source()) != 2 {
		t.Errorf("Source() = %d, want 2", len(te.Source()))
	}
}
