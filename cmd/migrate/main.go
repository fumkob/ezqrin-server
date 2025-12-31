package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Build database connection string from environment variables
	dbHost := getEnv("DB_HOST", "postgres")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "ezqrin")
	dbPassword := getEnv("DB_PASSWORD", "ezqrin_dev")
	dbName := getEnv("DB_NAME", "ezqrin_db")
	sslMode := getEnv("DB_SSL_MODE", "disable")

	databaseURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=%s",
		dbUser, dbPassword, dbHost, dbPort, dbName, sslMode,
	)

	// Migration files directory
	migrationsPath := "file://internal/infrastructure/database/migrations"

	// Create migrate instance
	m, err := migrate.New(migrationsPath, databaseURL)
	if err != nil {
		log.Fatalf("Failed to create migrate instance: %v", err)
	}
	defer m.Close()

	// Execute command
	command := os.Args[1]

	switch command {
	case "up":
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		log.Println("Migrations applied successfully")

	case "down":
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to rollback migrations: %v", err)
		}
		log.Println("Migrations rolled back successfully")

	case "step":
		if len(os.Args) < 3 {
			log.Fatal("Usage: migrate step <n>")
		}
		steps, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("Invalid step value: %v", err)
		}
		if err := m.Steps(steps); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Failed to run migration steps: %v", err)
		}
		log.Printf("Migration steps (%d) applied successfully\n", steps)

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			log.Fatalf("Failed to get migration version: %v", err)
		}
		fmt.Printf("Current version: %d\n", version)
		if dirty {
			fmt.Println("WARNING: Database is in dirty state")
		}

	case "force":
		if len(os.Args) < 3 {
			log.Fatal("Usage: migrate force <version>")
		}
		version, err := strconv.Atoi(os.Args[2])
		if err != nil {
			log.Fatalf("Invalid version value: %v", err)
		}
		if err := m.Force(version); err != nil {
			log.Fatalf("Failed to force version: %v", err)
		}
		log.Printf("Forced database version to %d\n", version)

	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: migrate <command> [args]")
	fmt.Println("\nCommands:")
	fmt.Println("  up              Apply all pending migrations")
	fmt.Println("  down            Rollback all migrations")
	fmt.Println("  step <n>        Apply next n migrations (use negative for rollback)")
	fmt.Println("  version         Show current migration version")
	fmt.Println("  force <version> Force set migration version (use with caution)")
	fmt.Println("\nEnvironment Variables:")
	fmt.Println("  DB_HOST     Database host (default: postgres)")
	fmt.Println("  DB_PORT     Database port (default: 5432)")
	fmt.Println("  DB_USER     Database user (default: ezqrin)")
	fmt.Println("  DB_PASSWORD Database password (default: ezqrin_dev)")
	fmt.Println("  DB_NAME     Database name (default: ezqrin_db)")
	fmt.Println("  DB_SSL_MODE SSL mode (default: disable)")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
