package tcaller

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func resetCache() {
	callerCache = sync.Map{}
}

func captureCurrentLine(t *testing.T) (Caller, int) {
	t.Helper()
	_, _, line, _ := runtime.Caller(0)
	return Current(), line + 1
}

func captureSkipLine(t *testing.T) (Caller, Caller, int, int) {
	t.Helper()
	_, _, line0, _ := runtime.Caller(0)
	direct := New(0)
	_, _, line1, _ := runtime.Caller(0)
	ancestor := New(1)
	return direct, ancestor, line0 + 1, line1 + 1
}

//go:noinline
func helperCurrent() Caller {
	return Current()
}

//go:noinline
func helperCurrentSkipOne() Caller {
	return New(1)
}

//go:noinline
func helperParent() Caller {
	return Parent()
}

type testService struct{}

func plainFunctionCaller() Caller {
	return Current()
}

func (testService) valueMethodCaller() Caller {
	return Current()
}

func (*testService) pointerMethodCaller() Caller {
	return Current()
}

func TestCurrentResolvesExpectedFunctionAndLine(t *testing.T) {
	resetCache()

	c, wantLine := captureCurrentLine(t)

	if c.IsZero() {
		t.Fatal("Current() returned zero caller")
	}
	if got := c.Function(); got != "github.com/tigorhutasuhut/telemetry-go/tcaller.captureCurrentLine" {
		t.Fatalf("Function() = %q", got)
	}
	if got := c.ShortFunction(); got != "captureCurrentLine" {
		t.Fatalf("ShortFunction() = %q", got)
	}
	if got := c.File(); filepath.Base(got) != "tcaller_test.go" {
		t.Fatalf("File() = %q", got)
	}
	if got := c.ShortFile(); got != "tcaller_test.go" {
		t.Fatalf("ShortFile() = %q", got)
	}
	if got := c.Line(); got != wantLine {
		t.Fatalf("Line() = %d, want %d", got, wantLine)
	}
	if got := c.FileLine(); got != "tcaller_test.go:"+strconv.Itoa(wantLine) {
		t.Fatalf("FileLine() = %q", got)
	}
	if got := c.String(); got != c.FileLine() {
		t.Fatalf("String() = %q, FileLine() = %q", got, c.FileLine())
	}
}

func TestCurrentSkipSemantics(t *testing.T) {
	resetCache()

	direct, ancestor, directLine, _ := captureSkipLine(t)

	if got := direct.Function(); got != "github.com/tigorhutasuhut/telemetry-go/tcaller.captureSkipLine" {
		t.Fatalf("direct.Function() = %q", got)
	}
	if got := direct.Line(); got != directLine {
		t.Fatalf("direct.Line() = %d, want %d", got, directLine)
	}
	if got := ancestor.Function(); got != "github.com/tigorhutasuhut/telemetry-go/tcaller.TestCurrentSkipSemantics" {
		t.Fatalf("ancestor.Function() = %q", got)
	}
}

func TestFromPCResolvesMetadata(t *testing.T) {
	resetCache()

	c := helperCurrent()
	from := FromPC(c.Uintptr())

	if from.IsZero() {
		t.Fatal("FromPC returned zero for valid pc")
	}
	if from.Function() != c.Function() {
		t.Fatalf("Function mismatch: %q != %q", from.Function(), c.Function())
	}
	if from.File() != c.File() {
		t.Fatalf("File mismatch: %q != %q", from.File(), c.File())
	}
	if from.Line() != c.Line() {
		t.Fatalf("Line mismatch: %d != %d", from.Line(), c.Line())
	}
}

func TestZeroCallerIsSafe(t *testing.T) {
	resetCache()

	var c Caller
	if !c.IsZero() {
		t.Fatal("zero caller should be zero")
	}
	if c.Uintptr() != 0 || c.File() != "" || c.ShortFile() != "" || c.Line() != 0 || c.Function() != "" || c.ShortFunction() != "" {
		t.Fatal("zero caller returned unexpected metadata")
	}
	if c.FileLine() != unknown || c.String() != unknown {
		t.Fatal("zero caller string formatting should be unknown")
	}
}

func TestInvalidCallerIsSafe(t *testing.T) {
	resetCache()

	c := FromPC(1)
	if c.IsZero() {
		t.Fatal("invalid non-zero caller should not report zero")
	}
	if c.File() != "" || c.ShortFile() != "" || c.Line() != 0 || c.Function() != "" || c.ShortFunction() != "" {
		t.Fatal("invalid caller returned unexpected metadata")
	}
	if c.FileLine() != unknown || c.String() != unknown {
		t.Fatal("invalid caller string formatting should be unknown")
	}
}

func TestShortFunctionFormatting(t *testing.T) {
	resetCache()

	plain := plainFunctionCaller()
	if got := plain.ShortFunction(); got != "plainFunctionCaller" {
		t.Fatalf("plain ShortFunction() = %q", got)
	}

	value := (testService{}).valueMethodCaller()
	if got := value.ShortFunction(); got != "testService.valueMethodCaller" {
		t.Fatalf("value method ShortFunction() = %q", got)
	}

	pointer := (&testService{}).pointerMethodCaller()
	if got := pointer.ShortFunction(); got != "(*testService).pointerMethodCaller" {
		t.Fatalf("pointer method ShortFunction() = %q", got)
	}
}

func TestFileLineAndStringFormatting(t *testing.T) {
	resetCache()

	c, wantLine := captureCurrentLine(t)
	want := "tcaller_test.go:" + strconv.Itoa(wantLine)
	if got := c.FileLine(); got != want {
		t.Fatalf("FileLine() = %q, want %q", got, want)
	}
	if got := c.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestShortFileFallbackWithoutCWDPrefix(t *testing.T) {
	got := trimCWD("/tmp/example/file.go")
	if got != "/tmp/example/file.go" {
		t.Fatalf("trimCWD() = %q", got)
	}
}

func TestMarshalHelpers(t *testing.T) {
	resetCache()

	c, wantLine := captureCurrentLine(t)
	want := "tcaller_test.go:" + strconv.Itoa(wantLine)

	text, err := c.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error = %v", err)
	}
	if string(text) != want {
		t.Fatalf("MarshalText() = %q, want %q", text, want)
	}

	data, err := c.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON() error = %v", err)
	}
	if string(data) != `"`+want+`"` {
		t.Fatalf("MarshalJSON() = %q", data)
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("json.Unmarshal error = %v", err)
	}
	if s != want {
		t.Fatalf("unmarshaled string = %q, want %q", s, want)
	}
}

func TestCurrentLargeSkipIsSafe(t *testing.T) {
	resetCache()

	c := New(1 << 20)
	if !c.IsZero() {
		t.Fatal("large skip should return zero caller")
	}
}

func TestCacheDoesNotChangeCorrectness(t *testing.T) {
	resetCache()

	c := helperCurrent()
	firstFunction := c.Function()
	firstFile := c.File()
	firstShortFile := c.ShortFile()
	firstLine := c.Line()
	firstShort := c.ShortFunction()
	firstFileLine := c.FileLine()

	for range 10 {
		if got := c.Function(); got != firstFunction {
			t.Fatalf("cached Function() = %q, want %q", got, firstFunction)
		}
		if got := c.File(); got != firstFile {
			t.Fatalf("cached File() = %q, want %q", got, firstFile)
		}
		if got := c.ShortFile(); got != firstShortFile {
			t.Fatalf("cached ShortFile() = %q, want %q", got, firstShortFile)
		}
		if got := c.Line(); got != firstLine {
			t.Fatalf("cached Line() = %d, want %d", got, firstLine)
		}
		if got := c.ShortFunction(); got != firstShort {
			t.Fatalf("cached ShortFunction() = %q, want %q", got, firstShort)
		}
		if got := c.FileLine(); got != firstFileLine {
			t.Fatalf("cached FileLine() = %q, want %q", got, firstFileLine)
		}
	}
}

func TestHelperSkipTargetsCaller(t *testing.T) {
	resetCache()

	c := helperCurrentSkipOne()
	if got := c.Function(); got != "github.com/tigorhutasuhut/telemetry-go/tcaller.TestHelperSkipTargetsCaller" {
		t.Fatalf("Function() = %q", got)
	}
}

func TestParentTargetsCallerParent(t *testing.T) {
	resetCache()

	c := helperParent()
	if got := c.Function(); got != "github.com/tigorhutasuhut/telemetry-go/tcaller.TestParentTargetsCallerParent" {
		t.Fatalf("Function() = %q", got)
	}
}

func TestCurrentFunctionNameLooksQualified(t *testing.T) {
	resetCache()

	c := helperCurrent()
	if got := c.Function(); !strings.Contains(got, "/tcaller.") {
		t.Fatalf("Function() = %q, want qualified name", got)
	}
}
