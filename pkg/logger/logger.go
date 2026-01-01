// Package logger provides structured logging with context support using zap.
//
// The logger supports different log levels (debug, info, warn, error) and formats
// (JSON for production, console for development). It integrates with context.Context
// to automatically track request IDs across service boundaries.
//
// Example usage:
//
//	// Initialize logger
//	cfg := logger.Config{
//		Level:       "info",
//		Format:      "json",
//		Environment: "production",
//	}
//	log, err := logger.New(cfg)
//	if err != nil {
//		panic(err)
//	}
//	defer log.Sync()
//
//	// Basic logging
//	log.Info("server started", zap.Int("port", 8080))
//	log.Error("database error", zap.Error(err))
//
//	// Context-aware logging
//	ctx := logger.ContextWithRequestID(context.Background(), "req-123")
//	log.WithContext(ctx).Info("processing request")
//
//	// Add structured fields
//	log.WithFields(
//		zap.String("user_id", userID),
//		zap.String("action", "login"),
//	).Info("user authenticated")
package logger

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// contextKey is the type for context keys to avoid collisions
type contextKey string

const (
	// requestIDKey is the context key for request ID
	requestIDKey contextKey = "request_id"
)

// Logger wraps zap.Logger to provide structured logging with context support
type Logger struct {
	*zap.Logger
}

// Config holds logger configuration
type Config struct {
	Level       string // debug, info, warn, error
	Format      string // json or text
	Environment string // development or production
}

// New creates a new Logger instance based on the provided configuration
func New(cfg Config) (*Logger, error) {
	var zapCfg zap.Config

	// Configure based on environment
	if cfg.Environment == "production" {
		zapCfg = zap.NewProductionConfig()
		zapCfg.EncoderConfig.TimeKey = "timestamp"
		zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	// Set log level
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}
	zapCfg.Level = zap.NewAtomicLevelAt(level)

	// Set output format
	if cfg.Format == "json" {
		zapCfg.Encoding = "json"
	} else {
		zapCfg.Encoding = "console"
	}

	// Build logger
	zapLogger, err := zapCfg.Build(
		zap.AddCallerSkip(1), // Skip wrapper function in stack trace
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return &Logger{Logger: zapLogger}, nil
}

// parseLevel converts string log level to zapcore.Level
func parseLevel(level string) (zapcore.Level, error) {
	switch level {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("invalid log level: %s", level)
	}
}

// WithContext returns a logger with request ID from context if present
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if requestID := GetRequestID(ctx); requestID != "" {
		return &Logger{
			Logger: l.Logger.With(zap.String("request_id", requestID)),
		}
	}
	return l
}

// WithRequestID returns a logger with the specified request ID
func (l *Logger) WithRequestID(requestID string) *Logger {
	return &Logger{
		Logger: l.Logger.With(zap.String("request_id", requestID)),
	}
}

// WithFields returns a logger with additional fields
func (l *Logger) WithFields(fields ...zap.Field) *Logger {
	return &Logger{
		Logger: l.Logger.With(fields...),
	}
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.Logger.Debug(msg, fields...)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.Logger.Info(msg, fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.Logger.Warn(msg, fields...)
}

// Error logs an error message
func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.Logger.Error(msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, fields...)
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}

// ContextWithRequestID returns a new context with the request ID
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDKey, requestID)
}

// GetRequestID retrieves the request ID from context.
// Returns empty string if context is nil or doesn't contain a request ID.
// This prevents panics when context is not properly initialized.
func GetRequestID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if requestID, ok := ctx.Value(requestIDKey).(string); ok {
		return requestID
	}
	return ""
}
