package tlog

import (
	"context"

	"github.com/tigorhutasuhut/telemetry-go/tcaller"
)

// Log builds an Entry and emits it via Print.
//
// Example:
//
//	tlog.Log(ctx,
//		tlog.Opt().
//			Message("request completed").
//			Fields("status", 200, "path", "/healthz"),
//	)
func Log(ctx context.Context, opts ...LogOption) {
	entry := New()
	for _, opt := range opts {
		if opt != nil {
			opt.applyLog(entry)
		}
	}
	if entry.caller == nil {
		if caller, ok := CallerFromContext(ctx); ok {
			entry.Caller(caller)
		} else {
			entry.Caller(tcaller.New(1))
		}
	}
	entry.Print(ctx)
}
