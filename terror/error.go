package terror

import (
	"log/slog"

	"github.com/tigorhutasuhut/telemetry-go/tcaller"
)

// Error is the public interface for an application-level error. It extends the
// standard [error] and [slog.LogValuer] interfaces with typed accessors and
// chainable mutators, so external packages can depend on the interface without
// coupling to the concrete implementation.
//
// The concrete implementation returned by [Wrap], [Fail], [Multi], and the
// variant helpers is an unexported *appError.
type Error interface {
	error
	slog.LogValuer

	// Unwrap supports [errors.Is] / [errors.As] chain traversal.
	Unwrap() []error

	// --- Getters ---

	// Message returns the human-readable error description.
	Message() string

	// Code returns the machine-readable error code, or nil if unset.
	Code() Code

	// Caller returns the source location that created or wrapped this error.
	Caller() tcaller.Caller

	// Fields returns the structured context fields attached to the error.
	// The returned slice must not be mutated; use [AddFields] to append.
	Fields() []any

	// Source returns the list of wrapped errors.
	Source() []error

	// --- Setters (all chainable, return Error interface) ---

	// SetMessage sets the human-readable error message.
	SetMessage(msg string) Error

	// SetCode sets the machine-readable error code.
	SetCode(code Code) Error

	// SetCaller overrides the recorded source location.
	SetCaller(caller tcaller.Caller) Error

	// AddFields appends structured context fields.
	AddFields(fields ...any) Error

	// SetSource replaces the wrapped-error slice wholesale.
	SetSource(src []error) Error

	// Join appends non-nil errors to the wrapped-error list.
	Join(errs ...error) Error

	// Resolve cleans nil entries from the wrapped-error list and returns the
	// error as a plain [error]. If the list is empty after cleaning, Resolve
	// returns nil — safe to use as the final return value when accumulating
	// errors with [Join].
	Resolve() error
}

// appError is the concrete, unexported implementation of [Error].
type appError struct {
	message string
	code    Code
	caller  tcaller.Caller
	fields  []any
	source  []error
}

// Verify at compile time that *appError satisfies Error.
var _ Error = (*appError)(nil)

// --- error / Unwrap ---

// Error implements the error interface.
//
// Priority:
//  1. message, if non-empty.
//  2. code.String(), if code is non-nil.
//  3. Empty string.
func (e *appError) Error() string {
	if e.message != "" {
		return e.message
	}
	if e.code != nil {
		return e.code.String()
	}
	return ""
}

// Unwrap implements the multi-error interface used by [errors.Is] and
// [errors.As].
func (e *appError) Unwrap() []error { return e.source }

// --- Getters ---

func (e *appError) Message() string        { return e.message }
func (e *appError) Code() Code             { return e.code }
func (e *appError) Caller() tcaller.Caller { return e.caller }
func (e *appError) Fields() []any          { return e.fields }
func (e *appError) Source() []error        { return e.source }

// --- Setters ---

func (e *appError) SetMessage(msg string) Error {
	e.message = msg
	return e
}

func (e *appError) SetCode(code Code) Error {
	e.code = code
	return e
}

func (e *appError) SetCaller(caller tcaller.Caller) Error {
	e.caller = caller
	return e
}

func (e *appError) AddFields(fields ...any) Error {
	e.fields = append(e.fields, fields...)
	return e
}

func (e *appError) SetSource(src []error) Error {
	e.source = src
	return e
}

// Join appends non-nil errors to e.source. It returns e for chaining.
func (e *appError) Join(errs ...error) Error {
	for _, err := range errs {
		if err != nil {
			e.source = append(e.source, err)
		}
	}
	return e
}

// Resolve cleans nil entries from e.source and returns e as an error.
// If source is empty after cleaning, Resolve returns nil.
func (e *appError) Resolve() error {
	cleaned := e.source[:0]
	for _, src := range e.source {
		if src != nil {
			cleaned = append(cleaned, src)
		}
	}
	e.source = cleaned
	if len(e.source) == 0 {
		return nil
	}
	return e
}
