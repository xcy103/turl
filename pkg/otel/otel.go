// Package otel wires OpenTelemetry distributed tracing. It builds a tracer
// provider that exports spans over OTLP/gRPC (e.g. to Jaeger) and registers it
// as the global provider, so instrumentation elsewhere needs no extra wiring.
//
// Instrumentation (gin, gorm, redis, manual spans) always runs; until Init
// installs a real provider it resolves to a no-op, so tracing is effectively
// gated by configuration alone.
package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/beihai0xff/turl/configs"
)

// ServiceName identifies this service in traces.
const ServiceName = "turl"

// ShutdownFunc flushes and stops the tracer provider.
type ShutdownFunc func(context.Context) error

// noopShutdown is returned when tracing is disabled.
func noopShutdown(context.Context) error { return nil }

// Init installs a global tracer provider and propagator from config. It returns
// a shutdown function that flushes pending spans on exit. When tracing is
// disabled or unconfigured it is a no-op.
func Init(ctx context.Context, c *configs.TracingConfig) (ShutdownFunc, error) {
	if c == nil || !c.Enable {
		return noopShutdown, nil
	}

	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(c.OTLPEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create otlp exporter: %w", err)
	}

	res, err := resource.New(ctx, resource.WithAttributes(semconv.ServiceName(ServiceName)))
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	ratio := c.SamplingRatio
	if ratio == 0 {
		ratio = configs.DefaultSamplingRatio
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// Tracer returns the named tracer from the global provider, used for manual spans.
func Tracer() trace.Tracer {
	return otel.Tracer(ServiceName)
}
