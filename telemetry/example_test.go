package telemetry_test

import (
	"context"

	"github.com/tigorhutasuhut/telemetry-go/telemetry"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func ExampleStartSpan() {
	ctx := context.Background()

	ctx, span := telemetry.StartSpan(ctx,
		telemetry.Span().
			Name("user.lookup").
			Kind(trace.SpanKindInternal).
			Attributes(
				attribute.String("user.id", "123"),
				attribute.String("component", "example"),
			),
	)
	defer span.End()

	_, child := telemetry.StartSpan(ctx, telemetry.Span().Name("user.lookup.db"))
	child.End()
}
