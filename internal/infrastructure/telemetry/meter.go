package telemetry

import (
	"context"
	"fmt"

	"github.com/fumkob/ezqrin-server/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

// NewMeterProvider creates a new MeterProvider.
// When cfg.Enabled is false, it returns a noop provider.
// The res parameter is the shared OTel resource (may be nil when disabled).
func NewMeterProvider(
	ctx context.Context,
	cfg config.TelemetryConfig,
	res *resource.Resource,
) (*metric.MeterProvider, error) {
	if !cfg.Enabled {
		return metric.NewMeterProvider(), nil
	}

	dialOpts := buildDialOptions(cfg)

	exporter, err := otlpmetricgrpc.New(ctx,
		otlpmetricgrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlpmetricgrpc.WithDialOption(dialOpts...),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric exporter: %w", err)
	}

	mp := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(exporter)),
		metric.WithResource(res),
	)

	return mp, nil
}
