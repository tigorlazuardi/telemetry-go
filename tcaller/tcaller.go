package tcaller

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

const unknown = "unknown"

// Caller represents a captured program counter.
//
// The zero value is intentional and safe. It represents an unknown caller and
// resolves to empty file/function values, line 0, and the string form
// "unknown".
type Caller uintptr

type metadata struct {
	file          string
	shortFile     string
	line          int
	function      string
	shortFunction string
	fileLine      string
}

var callerCache sync.Map
var cwd = currentWorkingDirectory()

// New returns the caller at the given stack depth.
//
// Skip semantics are defined relative to New itself:
//   - New(0) returns the direct caller of New
//   - New(1) returns that caller's caller
func New(skip int) Caller {
	if skip < 0 {
		skip = 0
	}

	pc, _, _, ok := runtime.Caller(skip + 1)
	if !ok || pc == 0 {
		return 0
	}

	return Caller(pc)
}

// Current returns the direct caller.
func Current() Caller {
	pc, _, _, ok := runtime.Caller(1)
	if !ok || pc == 0 {
		return 0
	}
	return Caller(pc)
}

// Parent returns the caller of the direct caller.
func Parent() Caller {
	pc, _, _, ok := runtime.Caller(2)
	if !ok || pc == 0 {
		return 0
	}
	return Caller(pc)
}

// FromPC wraps a program counter as a Caller.
func FromPC(pc uintptr) Caller {
	return Caller(pc)
}

// IsZero reports whether c is the zero caller.
func (c Caller) IsZero() bool { return c == 0 }

// Uintptr returns the raw program counter.
func (c Caller) Uintptr() uintptr { return uintptr(c) }

// File returns the full file path for the caller, or "" if unavailable.
func (c Caller) File() string { return c.resolve().file }

// ShortFile returns the file path with the current working directory prefix
// removed when present.
func (c Caller) ShortFile() string { return c.resolve().shortFile }

// Line returns the source line for the caller, or 0 if unavailable.
func (c Caller) Line() int { return c.resolve().line }

// Function returns the fully-qualified function name, or "" if unavailable.
func (c Caller) Function() string { return c.resolve().function }

// ShortFunction returns a shorter function name suitable for logs.
func (c Caller) ShortFunction() string { return c.resolve().shortFunction }

// FileLine returns the caller formatted as "basename.go:line".
// It returns "unknown" when the caller cannot be resolved.
func (c Caller) FileLine() string {
	m := c.resolve()
	if m.fileLine == "" {
		return unknown
	}
	return m.fileLine
}

// String returns the default string form of the caller.
func (c Caller) String() string { return c.FileLine() }

// MarshalText implements encoding.TextMarshaler.
func (c Caller) MarshalText() ([]byte, error) { return []byte(c.String()), nil }

// MarshalJSON implements json.Marshaler as a JSON string.
func (c Caller) MarshalJSON() ([]byte, error) { return json.Marshal(c.String()) }

func (c Caller) resolve() metadata {
	if c.IsZero() {
		return metadata{}
	}

	if cached, ok := callerCache.Load(c); ok {
		return cached.(metadata)
	}

	m := resolveMetadata(c)
	actual, _ := callerCache.LoadOrStore(c, m)
	return actual.(metadata)
}

func resolveMetadata(c Caller) metadata {
	pc := uintptr(c)
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return metadata{}
	}

	file, line := fn.FileLine(pc)
	function := fn.Name()

	m := metadata{
		file:     file,
		line:     line,
		function: function,
	}
	if file != "" {
		m.shortFile = trimCWD(file)
	}

	if function != "" {
		m.shortFunction = shortFunctionName(function)
	}

	if file != "" && line > 0 {
		m.fileLine = filepath.Base(file) + ":" + strconv.Itoa(line)
	}

	return m
}

func shortFunctionName(full string) string {
	if full == "" {
		return ""
	}

	lastSlash := strings.LastIndexByte(full, '/')
	searchFrom := lastSlash + 1
	if lastSlash < 0 {
		searchFrom = 0
	}

	firstDot := strings.IndexByte(full[searchFrom:], '.')
	if firstDot < 0 {
		return full[searchFrom:]
	}
	start := searchFrom + firstDot + 1
	if start >= len(full) {
		return ""
	}

	return full[start:]
}

func trimCWD(file string) string {
	if file == "" {
		return ""
	}
	if cwd == "" {
		return file
	}

	cleanFile := filepath.Clean(file)
	cleanCWD := filepath.Clean(cwd)
	if cleanFile == cleanCWD {
		return "."
	}

	prefix := cleanCWD + string(filepath.Separator)
	if strings.HasPrefix(cleanFile, prefix) {
		return strings.TrimPrefix(cleanFile, prefix)
	}

	return file
}

func currentWorkingDirectory() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}
