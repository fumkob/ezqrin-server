package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
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
	Port        int
	Environment string
}

// DatabaseConfig contains database connection configuration
type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// RedisConfig contains Redis connection configuration
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
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

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{}

	// Load Server configuration
	if err := loadServerConfig(&cfg.Server); err != nil {
		return nil, fmt.Errorf("server config: %w", err)
	}

	// Load Database configuration
	if err := loadDatabaseConfig(&cfg.Database); err != nil {
		return nil, fmt.Errorf("database config: %w", err)
	}

	// Load Redis configuration
	if err := loadRedisConfig(&cfg.Redis); err != nil {
		return nil, fmt.Errorf("redis config: %w", err)
	}

	// Load JWT configuration
	if err := loadJWTConfig(&cfg.JWT); err != nil {
		return nil, fmt.Errorf("jwt config: %w", err)
	}

	// Load Logging configuration
	if err := loadLoggingConfig(&cfg.Logging); err != nil {
		return nil, fmt.Errorf("logging config: %w", err)
	}

	// Load CORS configuration
	if err := loadCORSConfig(&cfg.CORS); err != nil {
		return nil, fmt.Errorf("cors config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	return cfg, nil
}

func loadServerConfig(cfg *ServerConfig) error {
	port, err := getEnvAsInt("SERVER_PORT", 8080)
	if err != nil {
		return err
	}
	cfg.Port = port

	cfg.Environment = getEnv("SERVER_ENV", "development")
	return nil
}

func loadDatabaseConfig(cfg *DatabaseConfig) error {
	cfg.Host = getEnv("DB_HOST", "localhost")

	port, err := getEnvAsInt("DB_PORT", 5432)
	if err != nil {
		return err
	}
	cfg.Port = port

	cfg.User = requireEnv("DB_USER")
	cfg.Password = requireEnv("DB_PASSWORD")
	cfg.Name = requireEnv("DB_NAME")
	cfg.SSLMode = getEnv("DB_SSL_MODE", "disable")

	maxOpenConns, err := getEnvAsInt("DB_MAX_OPEN_CONNS", 25)
	if err != nil {
		return err
	}
	cfg.MaxOpenConns = maxOpenConns

	maxIdleConns, err := getEnvAsInt("DB_MAX_IDLE_CONNS", 5)
	if err != nil {
		return err
	}
	cfg.MaxIdleConns = maxIdleConns

	connMaxLifetime, err := getEnvAsDuration("DB_CONN_MAX_LIFETIME", "5m")
	if err != nil {
		return err
	}
	cfg.ConnMaxLifetime = connMaxLifetime

	return nil
}

func loadRedisConfig(cfg *RedisConfig) error {
	cfg.Host = getEnv("REDIS_HOST", "localhost")

	port, err := getEnvAsInt("REDIS_PORT", 6379)
	if err != nil {
		return err
	}
	cfg.Port = port

	cfg.Password = getEnv("REDIS_PASSWORD", "")

	db, err := getEnvAsInt("REDIS_DB", 0)
	if err != nil {
		return err
	}
	cfg.DB = db

	return nil
}

func loadJWTConfig(cfg *JWTConfig) error {
	cfg.Secret = requireEnv("JWT_SECRET")

	accessExpiry, err := getEnvAsDuration("JWT_ACCESS_TOKEN_EXPIRY", "15m")
	if err != nil {
		return err
	}
	cfg.AccessTokenExpiry = accessExpiry

	refreshWebExpiry, err := getEnvAsDuration("JWT_REFRESH_TOKEN_EXPIRY_WEB", "168h")
	if err != nil {
		return err
	}
	cfg.RefreshTokenExpiryWeb = refreshWebExpiry

	refreshMobileExpiry, err := getEnvAsDuration("JWT_REFRESH_TOKEN_EXPIRY_MOBILE", "2160h")
	if err != nil {
		return err
	}
	cfg.RefreshTokenExpiryMobile = refreshMobileExpiry

	return nil
}

func loadLoggingConfig(cfg *LoggingConfig) error {
	cfg.Level = getEnv("LOG_LEVEL", "info")
	cfg.Format = getEnv("LOG_FORMAT", "json")
	return nil
}

func loadCORSConfig(cfg *CORSConfig) error {
	originsStr := getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000")
	cfg.AllowedOrigins = splitAndTrim(originsStr, ",")

	methodsStr := getEnv("CORS_ALLOWED_METHODS", "GET,POST,PUT,DELETE,OPTIONS")
	cfg.AllowedMethods = splitAndTrim(methodsStr, ",")

	headersStr := getEnv("CORS_ALLOWED_HEADERS", "Origin,Content-Type,Accept,Authorization")
	cfg.AllowedHeaders = splitAndTrim(headersStr, ",")

	allowCreds, err := getEnvAsBool("CORS_ALLOW_CREDENTIALS", true)
	if err != nil {
		return err
	}
	cfg.AllowCredentials = allowCreds

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate Server
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return fmt.Errorf("server port must be between 1 and 65535, got %d", c.Server.Port)
	}

	validEnvs := map[string]bool{"development": true, "production": true}
	if !validEnvs[c.Server.Environment] {
		return fmt.Errorf("server environment must be 'development' or 'production', got '%s'", c.Server.Environment)
	}

	// Validate Database
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.Port < 1 || c.Database.Port > 65535 {
		return fmt.Errorf("database port must be between 1 and 65535, got %d", c.Database.Port)
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
	if c.Database.MaxOpenConns < 1 {
		return fmt.Errorf("database max open connections must be at least 1, got %d", c.Database.MaxOpenConns)
	}
	if c.Database.MaxIdleConns < 0 {
		return fmt.Errorf("database max idle connections cannot be negative, got %d", c.Database.MaxIdleConns)
	}
	if c.Database.ConnMaxLifetime < 0 {
		return fmt.Errorf("database connection max lifetime cannot be negative")
	}

	// Validate Redis
	if c.Redis.Host == "" {
		return fmt.Errorf("redis host is required")
	}
	if c.Redis.Port < 1 || c.Redis.Port > 65535 {
		return fmt.Errorf("redis port must be between 1 and 65535, got %d", c.Redis.Port)
	}
	if c.Redis.DB < 0 {
		return fmt.Errorf("redis database index cannot be negative, got %d", c.Redis.DB)
	}

	// Validate JWT
	if c.JWT.Secret == "" {
		return fmt.Errorf("jwt secret is required")
	}
	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("jwt secret must be at least 32 characters, got %d", len(c.JWT.Secret))
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

	// Validate Logging
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[c.Logging.Level] {
		return fmt.Errorf("log level must be one of: debug, info, warn, error; got '%s'", c.Logging.Level)
	}

	validLogFormats := map[string]bool{"json": true, "text": true}
	if !validLogFormats[c.Logging.Format] {
		return fmt.Errorf("log format must be 'json' or 'text', got '%s'", c.Logging.Format)
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

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func requireEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		panic(fmt.Sprintf("required environment variable %s is not set", key))
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) (int, error) {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue, nil
	}

	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return 0, fmt.Errorf("invalid integer value for %s: %s", key, valueStr)
	}
	return value, nil
}

func getEnvAsBool(key string, defaultValue bool) (bool, error) {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue, nil
	}

	value, err := strconv.ParseBool(valueStr)
	if err != nil {
		return false, fmt.Errorf("invalid boolean value for %s: %s", key, valueStr)
	}
	return value, nil
}

func getEnvAsDuration(key, defaultValue string) (time.Duration, error) {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		valueStr = defaultValue
	}

	duration, err := time.ParseDuration(valueStr)
	if err != nil {
		return 0, fmt.Errorf("invalid duration value for %s: %s", key, valueStr)
	}
	return duration, nil
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
