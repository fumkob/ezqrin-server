// Package database provides PostgreSQL database connection and management utilities.
//
// This package implements connection pooling, transaction management, and health checking
// for PostgreSQL using pgx/v5. It follows Clean Architecture principles by residing in
// the infrastructure layer.
//
// Example usage:
//
//	cfg := &config.DatabaseConfig{
//		Host:            "localhost",
//		Port:            5432,
//		User:            "ezqrin",
//		Password:        "secret",
//		Name:            "ezqrin_db",
//		SSLMode:         "disable",
//		MaxConns:        25,
//		MinConns:        5,
//		ConnMaxLifetime: time.Hour,
//		ConnMaxIdleTime: 30 * time.Minute,
//	}
//
//	db, err := database.NewPostgresDB(ctx, cfg, logger)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer db.Close()
//
//	// Use the connection pool
//	pool := db.GetPool()
//	row := pool.QueryRow(ctx, "SELECT id, name FROM users WHERE id = $1", userID)
package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/fumkob/ezqrin-server/config"
	apperrors "github.com/fumkob/ezqrin-server/pkg/errors"
	"github.com/fumkob/ezqrin-server/pkg/logger"
)

// PostgresDB wraps pgxpool.Pool to provide database connection management
type PostgresDB struct {
	pool   *pgxpool.Pool
	logger *logger.Logger
}

// NewPostgresDB creates a new PostgreSQL connection pool with the provided configuration.
// It establishes a connection pool with custom settings for max connections, idle connections,
// and connection lifetime based on the provided config.
func NewPostgresDB(ctx context.Context, cfg *config.DatabaseConfig, log *logger.Logger) (*PostgresDB, error) {
	if ctx == nil {
		return nil, apperrors.Validation("context is required")
	}
	if cfg == nil {
		return nil, apperrors.Validation("database config is required")
	}
	if log == nil {
		return nil, apperrors.Validation("logger is required")
	}

	// Build connection string
	connString := buildConnectionString(cfg)

	// Parse connection string and configure pool
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, apperrors.Wrapf(err, "failed to parse database connection string")
	}

	// Configure connection pool settings
	poolConfig.MaxConns = int32(cfg.MaxConns)
	poolConfig.MinConns = int32(cfg.MinConns)
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	// Create connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, apperrors.Wrapf(err, "failed to create database connection pool")
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, apperrors.Wrapf(err, "failed to ping database")
	}

	log.WithContext(ctx).Info("database connection pool created",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.Name),
		zap.Int("max_conns", cfg.MaxConns),
		zap.Int("min_conns", cfg.MinConns),
	)

	return &PostgresDB{
		pool:   pool,
		logger: log,
	}, nil
}

// buildConnectionString constructs a PostgreSQL connection string from config
func buildConnectionString(cfg *config.DatabaseConfig) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.Name,
		cfg.SSLMode,
	)
}

// GetPool returns the underlying pgxpool.Pool for database operations
func (db *PostgresDB) GetPool() *pgxpool.Pool {
	return db.pool
}

// Close gracefully closes the database connection pool and releases resources.
// This should be called during application shutdown.
func (db *PostgresDB) Close() {
	if db.pool != nil {
		db.logger.Info("closing database connection pool")
		db.pool.Close()
	}
}

// Ping verifies the database connection is alive and returns an error if not.
// This is useful for health checks and connection validation.
func (db *PostgresDB) Ping(ctx context.Context) error {
	if err := db.pool.Ping(ctx); err != nil {
		return apperrors.Wrapf(err, "database ping failed")
	}
	return nil
}
