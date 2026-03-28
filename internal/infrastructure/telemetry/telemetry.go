package telemetry

import (
	"context"
	"errors"
	"fmt"

	"go.opentelemetry.io/otel"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// Providers holds the initialized OTel providers.
type Providers struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *metric.MeterProvider
	LoggerProvider *sdklog.LoggerProvider
}

// ShutdownFunc shuts down all telemetry providers.
type ShutdownFunc func(ctx context.Context) error

// Init initializes OpenTelemetry providers and registers them globally.
// Returns a Providers struct, a shutdown function, and any initialization error.
func Init(ctx context.Context, cfg Config) (*Providers, ShutdownFunc, error) {
	tp, err := NewTracerProvider(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init tracer provider: %w", err)
	}

	mp, err := NewMeterProvider(ctx, cfg)
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, nil, fmt.Errorf("failed to init meter provider: %w", err)
	}

	lp, err := NewLoggerProvider(ctx, cfg)
	if err != nil {
		_ = tp.Shutdown(ctx)
		_ = mp.Shutdown(ctx)
		return nil, nil, fmt.Errorf("failed to init logger provider: %w", err)
	}

	// Register globally
	otel.SetTracerProvider(tp)
	otel.SetMeterProvider(mp)

	providers := &Providers{
		TracerProvider: tp,
		MeterProvider:  mp,
		LoggerProvider: lp,
	}

	shutdown := func(ctx context.Context) error {
		return errors.Join(
			tp.Shutdown(ctx),
			mp.Shutdown(ctx),
			lp.Shutdown(ctx),
		)
	}

	return providers, shutdown, nil
}
