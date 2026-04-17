package terror

import "fmt"

// Wrapf wraps err with a message formatted from format and args, equivalent to
// calling [Wrap] with [Err().Message(fmt.Sprintf(format, args...))].
//
// Use [Wrap] with [Err] when you also need to set a [Code] or extra fields.
//
// The caller is automatically captured from the Wrapf call site.
func Wrapf(err error, format string, args ...any) Error {
	return &appError{
		caller:  resolveErrorCaller(nil),
		message: fmt.Sprintf(format, args...),
		source:  []error{err},
	}
}

// Wrapw wraps err with a plain message and slog-style structured fields,
// equivalent to calling [Wrap] with [Err().Message(msg).Fields(fields...)].
//
// fields follows the same key-value conventions as [ErrorOptions.Fields]:
// alternating string keys and arbitrary values, or [slog.Attr] elements.
//
// Use [Wrap] with [Err] when you also need to set a [Code].
//
// The caller is automatically captured from the Wrapw call site.
func Wrapw(err error, msg string, fields ...any) Error {
	return &appError{
		caller:  resolveErrorCaller(nil),
		message: msg,
		fields:  fields,
		source:  []error{err},
	}
}

// Failf creates a new [Error] with a message formatted from format and args,
// equivalent to calling [Fail] with [Err().Message(fmt.Sprintf(format, args...))].
//
// Use [Fail] with [Err] when you also need to set a [Code] or extra fields.
//
// The caller is automatically captured from the Failf call site.
func Failf(format string, args ...any) Error {
	return &appError{
		caller:  resolveErrorCaller(nil),
		message: fmt.Sprintf(format, args...),
	}
}

// Failw creates a new [Error] with a plain message and slog-style structured
// fields, equivalent to calling [Fail] with [Err().Message(msg).Fields(fields...)].
//
// fields follows the same key-value conventions as [ErrorOptions.Fields]:
// alternating string keys and arbitrary values, or [slog.Attr] elements.
//
// Use [Fail] with [Err] when you also need to set a [Code].
//
// The caller is automatically captured from the Failw call site.
func Failw(msg string, fields ...any) Error {
	return &appError{
		caller:  resolveErrorCaller(nil),
		message: msg,
		fields:  fields,
	}
}
