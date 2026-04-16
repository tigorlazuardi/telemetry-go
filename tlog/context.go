package tlog

import (
	"context"
	"log/slog"

	"github.com/tigorhutasuhut/telemetry-go/tcaller"
)

type (
	attrsContextKey  struct{}
	callerContextKey struct{}
)

// WithAttrs returns a child context with appended logging attrs.
func WithAttrs(ctx context.Context, attrs ...slog.Attr) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	current := AttrsFromContext(ctx)
	merged := make([]slog.Attr, 0, len(current)+len(attrs))
	merged = append(merged, current...)
	merged = append(merged, attrs...)
	return context.WithValue(ctx, attrsContextKey{}, merged)
}

// AttrsFromContext returns logging attrs from context.
func AttrsFromContext(ctx context.Context) []slog.Attr {
	if ctx == nil {
		return nil
	}
	attrs, _ := ctx.Value(attrsContextKey{}).([]slog.Attr)
	if len(attrs) == 0 {
		return nil
	}
	out := make([]slog.Attr, len(attrs))
	copy(out, attrs)
	return out
}

// WithCaller returns a child context with a logging caller.
func WithCaller(ctx context.Context, caller tcaller.Caller) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, callerContextKey{}, caller)
}

// WithCurrentCaller returns a child context with the current caller.
func WithCurrentCaller(ctx context.Context) context.Context {
	return WithCaller(ctx, tcaller.New(1))
}

// CallerFromContext returns the logging caller from context.
func CallerFromContext(ctx context.Context) (tcaller.Caller, bool) {
	if ctx == nil {
		return 0, false
	}
	caller, ok := ctx.Value(callerContextKey{}).(tcaller.Caller)
	return caller, ok
}
