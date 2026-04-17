// Package terror provides application-level error types with structured
// metadata for logging and tracing.
//
// Integration expectations:
//
//   - Use [Fail] to create a new error at the current callsite.
//   - Use [Wrap] to wrap an existing error, preserving the source chain.
//   - Errors carry a [Code] that encodes both an HTTP status and a string form.
//   - Errors carry a [tcaller.Caller] identifying where the error originated.
//   - Errors carry optional key-value fields for structured logging.
//   - Errors implement the multi-error interface via Unwrap() []error.
package terror
