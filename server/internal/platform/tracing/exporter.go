package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// ExporterConfig configures the OTLP trace exporter and tracer provider.
type ExporterConfig struct {
	Endpoint    string
	ServiceName string
	Environment string
	Insecure    bool
}

// NewProvider creates and installs a global OpenTelemetry tracer provider.
// The caller owns the returned provider and must call Shutdown during process
// shutdown so buffered spans are flushed.
func NewProvider(ctx context.Context, cfg ExporterConfig) (*sdktrace.TracerProvider, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("OTLP endpoint is required")
	}

	if cfg.ServiceName == "" {
		cfg.ServiceName = "leamout-server"
	}

	exporterOptions := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
	}

	if cfg.Insecure {
		exporterOptions = append(exporterOptions, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, exporterOptions...)
	if err != nil {
		return nil, fmt.Errorf("create OTLP trace exporter: %w", err)
	}

	attributes := []attribute.KeyValue{
		semconv.ServiceNameKey.String(cfg.ServiceName),
	}

	if cfg.Environment != "" {
		attributes = append(attributes, attribute.String("deployment.environment", cfg.Environment))
	}

	res, err := resource.New(
		ctx,
		resource.WithAttributes(attributes...),
	)
	if err != nil {
		_ = exporter.Shutdown(ctx)
		return nil, fmt.Errorf("create tracing resource: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return provider, nil
}
