package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const InstrumentationName = "github.com/leamout/leamout/server"

// Tracer provides Leamout's application-facing tracing API.
type Tracer struct {
	tracer trace.Tracer
}

// New creates a tracer using the supplied instrumentation name. If name is
// empty, Leamout's standard instrumentation name is used.
func New(name string) *Tracer {
	if name == "" {
		name = InstrumentationName
	}

	return &Tracer{
		tracer: otel.Tracer(name),
	}
}

// Start starts a new span and returns the derived context and span.
func (t *Tracer) Start(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.tracer.Start(ctx, name, opts...)
}

// Tracer returns the underlying OpenTelemetry tracer for integrations that
// need direct access to OpenTelemetry APIs.
func (t *Tracer) Tracer() trace.Tracer {
	return t.tracer
}
