package terror

import (
	"fmt"
	"log/slog"
)

// LogValue implements [slog.LogValuer] for [StatusCode].
// It returns the human-readable string form of the status code.
func (s StatusCode) LogValue() slog.Value {
	return slog.StringValue(s.String())
}

// LogValue implements [slog.LogValuer] for [Error].
//
// The returned group contains:
//   - "message"  — present only when [Error.Message] is non-empty.
//   - "code"     — the error code; delegated to its own LogValue if available.
//   - "caller"   — delegated to [tcaller.Caller.LogValue].
//   - "fields"   — structured key-value pairs from [Error.Fields], if any.
//   - "source"   — wrapped errors; omitted when empty, unwrapped when exactly
//     one, grouped with numeric keys "1", "2", … when more than one.
func (e *Error) LogValue() slog.Value {
	attrs := make([]slog.Attr, 0, 5)

	// message
	if e.Message != "" {
		attrs = append(attrs, slog.String("message", e.Message))
	}

	// code
	if e.Code != nil {
		if lv, ok := e.Code.(slog.LogValuer); ok {
			attrs = append(attrs, slog.Any("code", lv))
		} else {
			attrs = append(attrs, slog.String("code", e.Code.String()))
		}
	}

	// caller — Caller already implements slog.LogValuer
	attrs = append(attrs, slog.Any("caller", e.Caller))

	// fields
	if len(e.Fields) > 0 {
		fieldAttrs := argsToAttrs(e.Fields)
		if len(fieldAttrs) > 0 {
			attrs = append(attrs, slog.Attr{
				Key:   "fields",
				Value: slog.GroupValue(fieldAttrs...),
			})
		}
	}

	// source
	switch len(e.Source) {
	case 0:
		// omit
	case 1:
		attrs = append(attrs, slog.Any("source", errorLogValuer{e.Source[0]}))
	default:
		sourceAttrs := make([]slog.Attr, len(e.Source))
		for i, src := range e.Source {
			sourceAttrs[i] = slog.Any(fmt.Sprintf("%d", i+1), errorLogValuer{src})
		}
		attrs = append(attrs, slog.Attr{
			Key:   "source",
			Value: slog.GroupValue(sourceAttrs...),
		})
	}

	return slog.GroupValue(attrs...)
}

// errorLogValuer wraps an arbitrary error and implements [slog.LogValuer].
//
// If the wrapped error itself implements [slog.LogValuer], that implementation
// is delegated to directly. Otherwise a synthetic group with "type" and
// "message" fields is returned.
type errorLogValuer struct{ err error }

func (e errorLogValuer) LogValue() slog.Value {
	if lv, ok := e.err.(slog.LogValuer); ok {
		return lv.LogValue()
	}
	return slog.GroupValue(
		slog.String("type", fmt.Sprintf("%T", e.err)),
		slog.String("message", e.err.Error()),
	)
}

// argsToAttrs converts a variadic args slice (as used by slog) into a slice of
// [slog.Attr] values using the same rules as the standard library:
//   - An [slog.Attr] element is used as-is and consumes one slot.
//   - A string element followed by another element becomes the key of a new
//     Attr whose value is slog.AnyValue of the next element, consuming two slots.
//   - A string element with no following element is stored under the special key
//     "!BADKEY".
//   - Any other element is stored under "!BADKEY".
func argsToAttrs(args []any) []slog.Attr {
	attrs := make([]slog.Attr, 0, len(args))
	for len(args) > 0 {
		var a slog.Attr
		a, args = argsToAttr(args)
		attrs = append(attrs, a)
	}
	return attrs
}

func argsToAttr(args []any) (slog.Attr, []any) {
	switch x := args[0].(type) {
	case slog.Attr:
		return x, args[1:]
	case string:
		if len(args) < 2 {
			return slog.String("!BADKEY", x), args[1:]
		}
		return slog.Any(x, args[1]), args[2:]
	default:
		return slog.Any("!BADKEY", x), args[1:]
	}
}
