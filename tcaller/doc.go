// Package tcaller provides a lightweight caller value for logging and tracing.
//
// Integration expectations:
//
//   - Logging may attach an explicit caller with tlog.WithCaller(ctx, caller).
//   - Logging convenience helpers such as tlog.WithCurrentCaller(ctx) should use
//     Current() or New(skip) with an adjusted skip value at the callsite.
//   - When multiple caller sources exist, explicit caller values should take
//     precedence over inferred ones.
//   - Tracing should support explicit caller attachment via APIs such as
//     telemetry.Span().Caller(...).
//   - Tracing should not implicitly read caller values from context.
package tcaller
