package testutil

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/fumkob/ezqrin-server/config"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/cache"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/cache/redis"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/container"
	"github.com/fumkob/ezqrin-server/internal/infrastructure/database"
	"github.com/fumkob/ezqrin-server/internal/interface/api"
	"github.com/fumkob/ezqrin-server/pkg/logger"
	"github.com/gin-gonic/gin"
)

const (
	TestJWTSecret = "test-secret-key-minimum-32-characters-long-for-testing"
	TestDBName    = "ezqrin_test"

	testDBPort          = 5432
	testDBMaxConns      = 10
	testDBMinConns      = 2
	testDBMaxConnLife   = 30
	testDBMaxConnIdle   = 5
	testRedisPort       = 6379
	testRedisPoolSize   = 10
	testRedisMinIdle    = 2
	testRedisMaxRetries = 3
	testRedisDialSec    = 5
	testRedisRWSec      = 3
)

// NewTestConfig creates a standard test configuration from environment variables with sensible defaults.
func NewTestConfig() *config.Config {
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	redisDB := 1
	if s := os.Getenv("TEST_REDIS_DB"); s != "" {
		if n, err := strconv.Atoi(s); err == nil {
			redisDB = n
		}
	}

	return &config.Config{
		Database: config.DatabaseConfig{
			Host:            dbHost,
			Port:            testDBPort,
			User:            "ezqrin",
			Password:        "ezqrin_dev",
			Name:            TestDBName,
			SSLMode:         "disable",
			MaxConns:        testDBMaxConns,
			MinConns:        testDBMinConns,
			MaxConnLifetime: testDBMaxConnLife * time.Minute,
			MaxConnIdleTime: testDBMaxConnIdle * time.Minute,
		},
		Redis: config.RedisConfig{
			Host:         redisHost,
			Port:         testRedisPort,
			Password:     "",
			DB:           redisDB,
			PoolSize:     testRedisPoolSize,
			MinIdleConns: testRedisMinIdle,
			MaxRetries:   testRedisMaxRetries,
			DialTimeout:  testRedisDialSec * time.Second,
			ReadTimeout:  testRedisRWSec * time.Second,
			WriteTimeout: testRedisRWSec * time.Second,
		},
		JWT: config.JWTConfig{
			Secret: TestJWTSecret,
		},
		CORS: config.CORSConfig{
			AllowedOrigins: []string{"*"},
		},
	}
}

// NewTestLogger creates a logger that suppresses output during tests.
func NewTestLogger() *logger.Logger {
	log, _ := logger.New(logger.Config{
		Level:       "warn",
		Format:      "console",
		Environment: "development",
	})
	return log
}

// TestEnv holds all infrastructure needed for an integration test.
type TestEnv struct {
	Config      *config.Config
	Logger      *logger.Logger
	DB          database.Service
	Cache       cache.Service
	RedisClient *redis.Client
	Router      *gin.Engine
}

// NewTestEnv initializes all test infrastructure (DB, Redis, Router).
// Caller is responsible for calling Cleanup when done.
func NewTestEnv() (*TestEnv, error) {
	cfg := NewTestConfig()
	log := NewTestLogger()
	ctx := context.Background()

	db, err := database.NewPostgresDB(ctx, &cfg.Database, log)
	if err != nil {
		return nil, err
	}

	redisClient, err := redis.NewClient(&redis.ClientConfig{
		Host:         cfg.Redis.Host,
		Port:         strconv.Itoa(cfg.Redis.Port),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
		MaxRetries:   cfg.Redis.MaxRetries,
		DialTimeout:  cfg.Redis.DialTimeout,
		ReadTimeout:  cfg.Redis.ReadTimeout,
		WriteTimeout: cfg.Redis.WriteTimeout,
	})
	if err != nil {
		return nil, err
	}

	appContainer := container.NewContainer(cfg, log, db, redisClient)
	router := api.SetupRouter(&api.RouterDependencies{
		Config:    cfg,
		Logger:    log,
		DB:        db,
		Cache:     redisClient,
		Container: appContainer,
	})

	return &TestEnv{
		Config:      cfg,
		Logger:      log,
		DB:          db,
		Cache:       redisClient,
		RedisClient: redisClient,
		Router:      router,
	}, nil
}

// CleanDatabase truncates all data tables and flushes Redis.
// Call this in BeforeEach and/or AfterEach to ensure test isolation.
func CleanDatabase(db database.Service, redisClient *redis.Client) {
	ctx := context.Background()
	pool := db.GetPool()
	if _, err := pool.Exec(ctx, "TRUNCATE TABLE checkins, participants, events, users CASCADE"); err != nil {
		// Non-fatal: log but continue
		_ = err
	}
	if redisClient != nil {
		_ = redisClient.GetClient().FlushDB(ctx).Err()
	}
}
