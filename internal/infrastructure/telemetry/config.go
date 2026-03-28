// Package telemetry provides OpenTelemetry instrumentation for the application.
package telemetry

import (
	"os"
	"strconv"
)

const (
	defaultServiceName      = "ezqrin-server"
	defaultOTLPEndpoint     = "localhost:4317"
	defaultTracesSampler    = "always_on"
	defaultTracesSamplerArg = 1.0
	defaultLogsExporter     = "otlp"
)

// Config holds OpenTelemetry configuration for the telemetry subsystem.
type Config struct {
	Enabled          bool
	ServiceName      string
	OTLPEndpoint     string
	OTLPInsecure     bool
	TracesSampler    string
	TracesSamplerArg float64
	LogsExporter     string
}

// NewConfigFromEnv creates a Config by reading OTEL_* environment variables
// with sensible defaults.
func NewConfigFromEnv() Config {
	return Config{
		Enabled:          getEnvBool("OTEL_ENABLED", true),
		ServiceName:      getEnvString("OTEL_SERVICE_NAME", defaultServiceName),
		OTLPEndpoint:     getEnvString("OTEL_EXPORTER_OTLP_ENDPOINT", defaultOTLPEndpoint),
		OTLPInsecure:     getEnvBool("OTEL_EXPORTER_OTLP_INSECURE", true),
		TracesSampler:    getEnvString("OTEL_TRACES_SAMPLER", defaultTracesSampler),
		TracesSamplerArg: getEnvFloat64("OTEL_TRACES_SAMPLER_ARG", defaultTracesSamplerArg),
		LogsExporter:     getEnvString("OTEL_LOGS_EXPORTER", defaultLogsExporter),
	}
}

func getEnvString(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return defaultValue
	}
	return b
}

func getEnvFloat64(key string, defaultValue float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return defaultValue
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return defaultValue
	}
	return f
}
