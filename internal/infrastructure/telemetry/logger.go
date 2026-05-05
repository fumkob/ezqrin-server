package telemetry

import (
	"context"
	"fmt"

	"github.com/fumkob/ezqrin-server/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
)

// NewLoggerProvider creates a new LoggerProvider.
// When cfg.Enabled is false or LogsExporter is "none", it returns a noop provider.
// The res parameter is the shared OTel resource (may be nil when disabled).
func NewLoggerProvider(
	ctx context.Context,
	cfg config.TelemetryConfig,
	res *resource.Resource,
) (*sdklog.LoggerProvider, error) {
	if !cfg.Enabled || cfg.LogsExporter == "none" {
		return sdklog.NewLoggerProvider(), nil
	}

	opts := []otlploggrpc.Option{
		otlploggrpc.WithEndpoint(cfg.OTLPEndpoint),
	}
	if cfg.OTLPInsecure {
		opts = append(opts, otlploggrpc.WithInsecure())
	}

	exporter, err := otlploggrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create log exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)

	return lp, nil
}
