# tcaller tasks

## Goal
Implement `tcaller` as a lightweight caller utility package with `type Caller uintptr`, lazy metadata resolution, a global cache, and ergonomic helpers for logging and tracing.

## Task 1 — Finalize public API
**Goal:** Define the public API surface for `tcaller`.

**Status:** ✅ Completed

Deliverables:
- `type Caller uintptr`
- constructors:
	- `New(skip int) Caller`
	- `Current() Caller`
	- `Parent() Caller`
	- `FromPC(pc uintptr) Caller`
- methods:
  - `IsZero() bool`
  - `Uintptr() uintptr`
  - `File() string`
  - `ShortFile() string`
  - `Line() int`
  - `Function() string`
  - `ShortFunction() string`
  - `FileLine() string`
  - `String() string`
  - `MarshalText() ([]byte, error)`
  - `MarshalJSON() ([]byte, error)`

Acceptance criteria:
- API names are fixed.
- Zero-value behavior is documented and intentional.

## Task 2 — Implement caller capture
**Goal:** Capture a stable program counter from the current stack.

**Status:** ✅ Completed

Work:
- implement `Current(skip int) Caller`
	- implement `New(skip int) Caller`
	- implement `Current() Caller`
	- implement `Parent() Caller`
- implement `FromPC(pc uintptr) Caller`
	- define skip semantics so `New(0)` means the direct caller of `New`

Acceptance criteria:
- valid callsites are captured in normal code paths
- invalid/unavailable stack frames do not panic

## Task 3 — Implement lazy metadata resolution
**Goal:** Resolve file, line, and function information only when accessed.

**Status:** ✅ Completed

Work:
- build internal resolver from `Caller`
- use runtime APIs to resolve:
  - full file path
  - cwd-trimmed short file path
  - line number
  - full function name

Acceptance criteria:
- metadata is resolved lazily
- zero or invalid callers return safe empty values

## Task 4 — Add global cache
**Goal:** Avoid repeated runtime resolution for the same caller.

**Status:** ✅ Completed

Work:
- add a concurrency-safe global cache keyed by `Caller`
- cache resolved metadata:
  - file
  - line
  - function
  - short function
  - fileline

Acceptance criteria:
- repeated metadata lookups reuse cached resolution
- concurrent access is safe

## Task 5 — Implement formatting helpers
**Goal:** Provide ergonomic string helpers for logs and span naming.

**Status:** ✅ Completed

Work:
- implement `FileLine()` as `basename.go:line`
- implement `String()` to default to `FileLine()`
- fallback to `"unknown"` if unresolved
- implement `ShortFunction()` by trimming package path while preserving receiver and method name

Examples:
- `main.main` -> `main`
- `github.com/acme/x.Handle` -> `Handle`
- `github.com/acme/x.(*Service).Run` -> `(*Service).Run`

Acceptance criteria:
- formatting output is stable and human-readable
- short function naming matches the documented rules

## Task 6 — Implement marshaling helpers
**Goal:** Make `Caller` safe and ergonomic when serialized.

**Status:** ✅ Completed

Work:
- implement `MarshalText()` using `String()`
- implement `MarshalJSON()` as a JSON string using `String()`

Acceptance criteria:
- marshaling never panics
- zero/unresolved caller marshals predictably

## Task 7 — Add unit tests
**Goal:** Validate correctness and edge cases.

**Status:** ✅ Completed

Test coverage:
	- `Current()` resolves expected function
- skip offsets behave correctly
- `FromPC` resolves metadata for valid PCs
- zero caller is safe
- invalid caller is safe
- `ShortFunction()` formatting for plain functions, methods, and pointer receivers
- `FileLine()` and `String()` output formatting
- `MarshalText()` and `MarshalJSON()` output
- cache does not change correctness

Acceptance criteria:
- tests cover all public API methods
- no panics in zero/invalid cases

## Task 8 — Add benchmarks
**Goal:** Measure runtime cost and validate cache usefulness.

**Status:** ✅ Completed

Benchmarks:
	- `Current()`
	- `New(skip)`
- first metadata resolution
- cached metadata resolution
- `ShortFunction()` access

Acceptance criteria:
- benchmark suite exists
- results are enough to guide later optimization if needed

## Task 9 — Document integration expectations
**Goal:** Make `tcaller` usage clear for the rest of the library.

**Status:** ✅ Completed

Document:
- `tlog.WithCaller(ctx, caller)` usage
- `tlog.WithCurrentCaller(ctx)` usage
- logging caller precedence
- tracing explicit caller usage via `telemetry.Span().Caller(...)`
- tracing does not read caller from context

Acceptance criteria:
- expected interactions with logging and tracing are written down
- no ambiguity around caller precedence or context behavior
