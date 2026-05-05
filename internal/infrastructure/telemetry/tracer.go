package telemetry

import (
	"context"
	"fmt"

	"github.com/fumkob/ezqrin-server/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// NewTracerProvider creates a new TracerProvider.
// When cfg.Enabled is false, it returns a noop provider.
// The res parameter is the shared OTel resource (may be nil when disabled).
func NewTracerProvider(
	ctx context.Context,
	cfg config.TelemetryConfig,
	res *resource.Resource,
) (*sdktrace.TracerProvider, error) {
	if !cfg.Enabled {
		return sdktrace.NewTracerProvider(), nil
	}

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
	}
	if cfg.OTLPInsecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	exporter, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	sampler := resolveSampler(cfg.TracesSampler, cfg.TracesSamplerArg)

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	return tp, nil
}

// resolveSampler maps sampler name to sdktrace.Sampler.
func resolveSampler(name string, arg float64) sdktrace.Sampler {
	switch name {
	case "always_off":
		return sdktrace.NeverSample()
	case "traceidratio":
		return sdktrace.TraceIDRatioBased(arg)
	default: // "always_on"
		return sdktrace.AlwaysSample()
	}
}
