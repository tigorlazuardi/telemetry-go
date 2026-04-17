package terror

import "github.com/tigorhutasuhut/telemetry-go/tcaller"

// Error is an application-level error that carries a human-readable message,
// a machine-readable [Code], the source location that produced the error, and
// optional structured fields and wrapped causes.
type Error struct {
	// Message is a human-readable description of what went wrong.
	Message string

	// Code categorises the error (HTTP status, string label).
	Code Code

	// Caller is the source location that created or wrapped this error.
	Caller tcaller.Caller

	// Fields carries arbitrary structured context attached to the error.
	Fields []any

	// Source lists the wrapped errors. Use [errors.Is] / [errors.As] to
	// inspect the chain.
	Source []error
}

// Error implements the error interface.
//
// Priority:
//  1. Message, if non-empty.
//  2. Code.String(), if Code is non-nil.
//  3. Empty string.
func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Code != nil {
		return e.Code.String()
	}
	return ""
}

// Unwrap implements the multi-error interface used by [errors.Is] and
// [errors.As].
func (e *Error) Unwrap() []error {
	return e.Source
}
