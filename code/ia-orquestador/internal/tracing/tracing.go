// Package tracing configures the OpenTelemetry TracerProvider.
// Supported exporters: none | stdout | otlp | both
package tracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// Setup initializes the global TracerProvider and returns a shutdown function.
//   - exporter: "none" | "stdout" | "otlp" | "both"
//   - endpoint: OTLP HTTP endpoint, e.g. "localhost:4318" (used for "otlp" and "both")
func Setup(ctx context.Context, serviceName, exporter, endpoint string) (shutdown func(context.Context) error, err error) {
	if exporter == "none" || exporter == "" {
		// No-op: leaves the default no-op provider in place.
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(semconv.ServiceNameKey.String(serviceName)),
	)
	if err != nil {
		return nil, fmt.Errorf("tracing resource: %w", err)
	}

	opts := []sdktrace.TracerProviderOption{sdktrace.WithResource(res)}

	if exporter == "stdout" || exporter == "both" {
		exp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, fmt.Errorf("stdout exporter: %w", err)
		}
		opts = append(opts, sdktrace.WithBatcher(exp))
	}

	if exporter == "otlp" || exporter == "both" {
		exp, err := otlptracehttp.New(ctx,
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithInsecure(),
		)
		if err != nil {
			return nil, fmt.Errorf("OTLP exporter: %w", err)
		}
		opts = append(opts, sdktrace.WithBatcher(exp))
	}

	if exporter != "stdout" && exporter != "otlp" && exporter != "both" {
		return nil, fmt.Errorf("unknown exporter %q; valid: none | stdout | otlp | both", exporter)
	}

	tp := sdktrace.NewTracerProvider(opts...)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}
