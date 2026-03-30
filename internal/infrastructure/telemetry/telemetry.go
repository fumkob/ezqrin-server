package telemetry

import (
	"context"
	"errors"
	"fmt"

	"github.com/fumkob/ezqrin-server/config"
	"go.opentelemetry.io/otel"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Providers holds the initialized OTel providers.
type Providers struct {
	TracerProvider *sdktrace.TracerProvider
	MeterProvider  *metric.MeterProvider
	LoggerProvider *sdklog.LoggerProvider
}

// ShutdownFunc shuts down all telemetry providers.
type ShutdownFunc func(ctx context.Context) error

// buildDialOptions returns gRPC dial options based on configuration.
func buildDialOptions(cfg config.TelemetryConfig) []grpc.DialOption {
	if cfg.OTLPInsecure {
		return []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	}
	return nil
}

// buildResource creates an OTel resource with the service name attribute.
func buildResource(ctx context.Context, serviceName string) (*resource.Resource, error) {
	return resource.New(ctx, resource.WithAttributes(semconv.ServiceNameKey.String(serviceName)))
}

// Init initializes OpenTelemetry providers and registers them globally.
// Returns a Providers struct, a shutdown function, and any initialization error.
func Init(ctx context.Context, cfg config.TelemetryConfig) (*Providers, ShutdownFunc, error) {
	// Build shared resource once for all providers
	var res *resource.Resource
	if cfg.Enabled {
		var err error
		res, err = buildResource(ctx, cfg.ServiceName)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create resource: %w", err)
		}
	}

	tp, err := NewTracerProvider(ctx, cfg, res)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to init tracer provider: %w", err)
	}

	mp, err := NewMeterProvider(ctx, cfg, res)
	if err != nil {
		_ = tp.Shutdown(ctx)
		return nil, nil, fmt.Errorf("failed to init meter provider: %w", err)
	}

	lp, err := NewLoggerProvider(ctx, cfg, res)
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
