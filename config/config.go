// Package config provides hierarchical configuration management using Viper.
package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

const (
	envKeyValueParts   = 2
	jwtSecretMinLength = 32
	minPort            = 1
	maxPort            = 65535
	minDatabaseConns   = 1
	minRedisDB         = 0
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	JWT      JWTConfig
	Logging  LoggingConfig
	CORS     CORSConfig
}

// ServerConfig contains server-related configuration
type ServerConfig struct {
	Port         int
	Environment  string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DatabaseConfig contains database connection configuration
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxConns        int           // Maximum connections in pool (maps to pgxpool.MaxConns)
	MinConns        int           // Minimum connections to maintain in pool (maps to pgxpool.MinConns)
	MaxConnLifetime time.Duration // Maximum lifetime of a connection (maps to pgxpool.MaxConnLifetime)
	MaxConnIdleTime time.Duration // Maximum idle time of a connection (maps to pgxpool.MaxConnIdleTime)
}

// RedisConfig contains Redis connection configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int

	// Connection pool configuration
	PoolSize     int
	MinIdleConns int
	MaxRetries   int

	// Timeout configuration
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// JWTConfig contains JWT token configuration
type JWTConfig struct {
	Secret                   string
	AccessTokenExpiry        time.Duration
	RefreshTokenExpiryWeb    time.Duration
	RefreshTokenExpiryMobile time.Duration
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string
	Format string
}

// CORSConfig contains CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
}

// Load reads configuration from YAML files and environment variables.
// Configuration is loaded in the following priority (highest to lowest):
//  1. Environment variables (secrets and overrides)
//  2. Environment-specific YAML file (development.yaml or production.yaml)
//  3. Default YAML file (default.yaml)
func Load() (*Config, error) {
	v := viper.New()

	// Load .env.secrets file if it exists (for local development)
	_ = loadEnvFile(v, ".env.secrets")

	// Determine environment
	environment := getEnvOrDefault("SERVER_ENV", "development")

	// Load default.yaml (base configuration)
	v.SetConfigName("default")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.AddConfigPath("../config")
	v.AddConfigPath("../../config")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read default.yaml: %w", err)
	}

	// Merge environment-specific YAML
	v.SetConfigName(environment)
	_ = v.MergeInConfig() // Not an error if file doesn't exist

	// Bind environment variables AFTER loading YAML to ensure env vars take precedence
	bindEnvVars(v)

	// Map viper config to Config struct
	cfg := &Config{}
	if err := unmarshalConfig(v, cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return cfg, nil
}

// loadEnvFile loads environment variables from a file
func loadEnvFile(v *viper.Viper, filepath string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Log error but don't fail since we're only reading
			fmt.Fprintf(os.Stderr, "Warning: failed to close file %s: %v\n", filepath, err)
		}
	}()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", envKeyValueParts)
		if len(parts) != envKeyValueParts {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		viperKey := convertEnvKeyToViperKey(key)
		v.Set(viperKey, value)
	}

	return scanner.Err()
}

// envKeyMap defines the mapping between environment variables and viper configuration keys.
// This is the single source of truth for all environment variable bindings.
var envKeyMap = map[string]string{
	// Server
	"SERVER_PORT":          "server.port",
	"SERVER_ENV":           "server.environment",
	"SERVER_READ_TIMEOUT":  "server.read_timeout",
	"SERVER_WRITE_TIMEOUT": "server.write_timeout",
	"SERVER_IDLE_TIMEOUT":  "server.idle_timeout",

	// Database
	"DB_HOST":               "database.host",
	"DB_PORT":               "database.port",
	"DB_USER":               "database.user",
	"DB_PASSWORD":           "database.password",
	"DB_NAME":               "database.name",
	"DB_SSL_MODE":           "database.ssl_mode",
	"DB_MAX_CONNS":          "database.max_conns",
	"DB_MIN_CONNS":          "database.min_conns",
	"DB_MAX_CONN_LIFETIME":  "database.max_conn_lifetime",
	"DB_MAX_CONN_IDLE_TIME": "database.max_conn_idle_time",

	// Redis
	"REDIS_HOST":     "redis.host",
	"REDIS_PORT":     "redis.port",
	"REDIS_PASSWORD": "redis.password",
	"REDIS_DB":       "redis.db",

	// JWT
	"JWT_SECRET":                      "jwt.secret",
	"JWT_ACCESS_TOKEN_EXPIRY":         "jwt.access_token_expiry",
	"JWT_REFRESH_TOKEN_EXPIRY_WEB":    "jwt.refresh_token_expiry_web",
	"JWT_REFRESH_TOKEN_EXPIRY_MOBILE": "jwt.refresh_token_expiry_mobile",

	// Logging
	"LOG_LEVEL":  "logging.level",
	"LOG_FORMAT": "logging.format",

	// CORS
	"CORS_ALLOWED_ORIGINS":   "cors.allowed_origins",
	"CORS_ALLOWED_METHODS":   "cors.allowed_methods",
	"CORS_ALLOWED_HEADERS":   "cors.allowed_headers",
	"CORS_ALLOW_CREDENTIALS": "cors.allow_credentials",
}

// convertEnvKeyToViperKey converts environment variable key to viper key
func convertEnvKeyToViperKey(envKey string) string {
	if viperKey, ok := envKeyMap[envKey]; ok {
		return viperKey
	}
	return strings.ToLower(strings.ReplaceAll(envKey, "_", "."))
}

// bindEnvVars explicitly binds environment variables to viper keys
func bindEnvVars(v *viper.Viper) {
	for envVar, viperKey := range envKeyMap {
		_ = v.BindEnv(viperKey, envVar)
	}
}

// unmarshalRedisConfig maps Redis configuration from viper to Config
func unmarshalRedisConfig(v *viper.Viper, cfg *Config) {
	cfg.Redis.Host = v.GetString("redis.host")
	cfg.Redis.Port = v.GetInt("redis.port")
	cfg.Redis.Password = v.GetString("redis.password")
	cfg.Redis.DB = v.GetInt("redis.db")
	cfg.Redis.PoolSize = v.GetInt("redis.pool_size")
	cfg.Redis.MinIdleConns = v.GetInt("redis.min_idle_conns")
	cfg.Redis.MaxRetries = v.GetInt("redis.max_retries")
	cfg.Redis.DialTimeout = v.GetDuration("redis.dial_timeout")
	cfg.Redis.ReadTimeout = v.GetDuration("redis.read_timeout")
	cfg.Redis.WriteTimeout = v.GetDuration("redis.write_timeout")
}

// unmarshalConfig maps viper configuration to Config struct
func unmarshalConfig(v *viper.Viper, cfg *Config) error {
	cfg.Server.Port = v.GetInt("server.port")
	cfg.Server.Environment = v.GetString("server.environment")
	cfg.Server.ReadTimeout = v.GetDuration("server.read_timeout")
	cfg.Server.WriteTimeout = v.GetDuration("server.write_timeout")
	cfg.Server.IdleTimeout = v.GetDuration("server.idle_timeout")

	cfg.Database.Host = v.GetString("database.host")
	cfg.Database.Port = v.GetInt("database.port")
	cfg.Database.User = v.GetString("database.user")
	cfg.Database.Password = v.GetString("database.password")
	// Check environment variable first for database.name to support test database override
	cfg.Database.Name = getEnvOrDefault("DB_NAME", v.GetString("database.name"))
	cfg.Database.SSLMode = v.GetString("database.ssl_mode")
	cfg.Database.MaxConns = v.GetInt("database.max_conns")
	cfg.Database.MinConns = v.GetInt("database.min_conns")
	cfg.Database.MaxConnLifetime = v.GetDuration("database.max_conn_lifetime")
	cfg.Database.MaxConnIdleTime = v.GetDuration("database.max_conn_idle_time")

	unmarshalRedisConfig(v, cfg)

	cfg.JWT.Secret = v.GetString("jwt.secret")
	cfg.JWT.AccessTokenExpiry = v.GetDuration("jwt.access_token_expiry")
	cfg.JWT.RefreshTokenExpiryWeb = v.GetDuration("jwt.refresh_token_expiry_web")
	cfg.JWT.RefreshTokenExpiryMobile = v.GetDuration("jwt.refresh_token_expiry_mobile")

	cfg.Logging.Level = v.GetString("logging.level")
	cfg.Logging.Format = v.GetString("logging.format")

	// Handle CORS allowed_origins - can be string (comma-separated) or slice
	if originsStr := v.GetString("cors.allowed_origins"); originsStr != "" {
		// Environment variable format: comma-separated string
		cfg.CORS.AllowedOrigins = splitAndTrim(originsStr, ",")
	} else {
		// YAML format: array
		cfg.CORS.AllowedOrigins = v.GetStringSlice("cors.allowed_origins")
	}

	cfg.CORS.AllowedMethods = v.GetStringSlice("cors.allowed_methods")
	cfg.CORS.AllowedHeaders = v.GetStringSlice("cors.allowed_headers")
	cfg.CORS.AllowCredentials = v.GetBool("cors.allow_credentials")

	// Validate required fields
	if cfg.Database.User == "" {
		return fmt.Errorf("database user is required (set DB_USER)")
	}
	if cfg.Database.Password == "" {
		return fmt.Errorf("database password is required (set DB_PASSWORD)")
	}
	if cfg.Database.Name == "" {
		return fmt.Errorf("database name is required (set DB_NAME)")
	}
	if cfg.JWT.Secret == "" {
		return fmt.Errorf("jwt secret is required (set JWT_SECRET)")
	}

	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if err := c.validateServer(); err != nil {
		return err
	}
	if err := c.validateDatabase(); err != nil {
		return err
	}
	if err := c.validateRedis(); err != nil {
		return err
	}
	if err := c.validateJWT(); err != nil {
		return err
	}
	if err := c.validateLogging(); err != nil {
		return err
	}
	return nil
}

// GetDatabaseDSN returns the PostgreSQL connection string
func (c *Config) GetDatabaseDSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Database.Host,
		c.Database.Port,
		c.Database.User,
		c.Database.Password,
		c.Database.Name,
		c.Database.SSLMode,
	)
}

// GetRedisAddr returns the Redis connection address
func (c *Config) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}

// IsProduction returns true if running in production environment
func (c *Config) IsProduction() bool {
	return c.Server.Environment == "production"
}

// validateServer validates server configuration.
func (c *Config) validateServer() error {
	if c.Server.Port < minPort || c.Server.Port > maxPort {
		return fmt.Errorf("server port must be between %d and %d, got %d", minPort, maxPort, c.Server.Port)
	}

	validEnvs := map[string]bool{"development": true, "production": true, "test": true}
	if !validEnvs[c.Server.Environment] {
		return fmt.Errorf(
			"server environment must be 'development', 'production' or 'test', got '%s'",
			c.Server.Environment,
		)
	}

	if c.Server.ReadTimeout <= 0 {
		return fmt.Errorf("server read timeout must be positive")
	}
	if c.Server.WriteTimeout <= 0 {
		return fmt.Errorf("server write timeout must be positive")
	}
	if c.Server.IdleTimeout <= 0 {
		return fmt.Errorf("server idle timeout must be positive")
	}
	return nil
}

// validateDatabase validates database configuration.
func (c *Config) validateDatabase() error {
	if err := c.validateDatabaseConnection(); err != nil {
		return err
	}
	return c.validateDatabasePool()
}

// validateDatabaseConnection validates database connection parameters.
func (c *Config) validateDatabaseConnection() error {
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.Port < minPort || c.Database.Port > maxPort {
		return fmt.Errorf("database port must be between %d and %d, got %d", minPort, maxPort, c.Database.Port)
	}
	if c.Database.User == "" {
		return fmt.Errorf("database user is required")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("database password is required")
	}
	if c.Database.Name == "" {
		return fmt.Errorf("database name is required")
	}
	return nil
}

// validateDatabasePool validates database connection pool parameters.
func (c *Config) validateDatabasePool() error {
	if c.Database.MaxConns < minDatabaseConns {
		return fmt.Errorf(
			"database max open connections must be at least %d, got %d",
			minDatabaseConns,
			c.Database.MaxConns,
		)
	}
	if c.Database.MinConns < minRedisDB {
		return fmt.Errorf("database min connections cannot be negative, got %d", c.Database.MinConns)
	}
	if c.Database.MinConns > c.Database.MaxConns {
		return fmt.Errorf(
			"database min connections (%d) cannot exceed max connections (%d)",
			c.Database.MinConns,
			c.Database.MaxConns,
		)
	}
	if c.Database.MaxConnLifetime < 0 {
		return fmt.Errorf("database connection max lifetime cannot be negative")
	}
	if c.Database.MaxConnIdleTime < 0 {
		return fmt.Errorf("database connection max idle time cannot be negative")
	}
	return nil
}

// validateRedis validates Redis configuration.
func (c *Config) validateRedis() error {
	if c.Redis.Host == "" {
		return fmt.Errorf("redis host is required")
	}
	if c.Redis.Port < minPort || c.Redis.Port > maxPort {
		return fmt.Errorf("redis port must be between %d and %d, got %d", minPort, maxPort, c.Redis.Port)
	}
	if c.Redis.DB < minRedisDB {
		return fmt.Errorf("redis database index cannot be negative, got %d", c.Redis.DB)
	}
	return nil
}

// validateJWT validates JWT configuration.
func (c *Config) validateJWT() error {
	if c.JWT.Secret == "" {
		return fmt.Errorf("jwt secret is required")
	}
	if len(c.JWT.Secret) < jwtSecretMinLength {
		return fmt.Errorf("jwt secret must be at least %d characters, got %d", jwtSecretMinLength, len(c.JWT.Secret))
	}
	if c.JWT.AccessTokenExpiry <= 0 {
		return fmt.Errorf("jwt access token expiry must be positive")
	}
	if c.JWT.RefreshTokenExpiryWeb <= 0 {
		return fmt.Errorf("jwt refresh token expiry (web) must be positive")
	}
	if c.JWT.RefreshTokenExpiryMobile <= 0 {
		return fmt.Errorf("jwt refresh token expiry (mobile) must be positive")
	}
	return nil
}

// validateLogging validates logging configuration.
func (c *Config) validateLogging() error {
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[c.Logging.Level] {
		return fmt.Errorf(
			"log level must be one of: debug, info, warn, error; got '%s'",
			c.Logging.Level,
		)
	}

	validLogFormats := map[string]bool{"json": true, "text": true}
	if !validLogFormats[c.Logging.Format] {
		return fmt.Errorf("log format must be 'json' or 'text', got '%s'", c.Logging.Format)
	}
	return nil
}
