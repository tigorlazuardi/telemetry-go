package tlog_test

import (
	"context"

	"github.com/tigorhutasuhut/telemetry-go/tlog"
)

func ExampleLog() {
	ctx := context.Background()

	tlog.Log(ctx,
		tlog.Opt().
			Message("request completed").
			Fields("status", 200, "path", "/healthz"),
	)
}
