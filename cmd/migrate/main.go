package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/fumkob/ezqrin-server/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Load configuration using centralized config package
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Build database connection string from config
	databaseURL := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	// Migration files directory
	migrationsPath := "file://internal/infrastructure/database/migrations"

	// Create migrate instance
	m, migErr := migrate.New(migrationsPath, databaseURL)
	if migErr != nil {
		log.Fatalf("Failed to create migrate instance: %v", migErr)
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
	fmt.Println("\nConfiguration:")
	fmt.Println("  Database configuration is loaded from:")
	fmt.Println("  1. config/default.yaml (base configuration)")
	fmt.Println("  2. config/development.yaml or config/production.yaml (environment-specific)")
	fmt.Println("  3. Environment variables (DB_USER, DB_PASSWORD, DB_NAME, etc.)")
	fmt.Println("\n  See .env.secrets.example for required environment variables.")
}
