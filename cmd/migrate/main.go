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

const (
	minArgsForCommand     = 2
	minArgsForStepOrForce = 3
	migrationsPathPrefix  = "file://internal/infrastructure/database/migrations"
)

func main() {
	if len(os.Args) < minArgsForCommand {
		printUsage()
		os.Exit(1)
	}

	// Load configuration using centralized config package
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Build database connection string from config
	databaseURL := buildDatabaseURL(cfg)

	// Create migrate instance
	m, migErr := migrate.New(migrationsPathPrefix, databaseURL)
	if migErr != nil {
		log.Fatalf("Failed to create migrate instance: %v", migErr)
	}
	defer func() {
		if _, err := m.Close(); err != nil {
			log.Printf("Warning: failed to close migrate instance: %v", err)
		}
	}()

	// Execute command
	command := os.Args[1]
	if err := executeCommand(m, command); err != nil {
		log.Fatalf("Command failed: %v", err)
	}
}

// buildDatabaseURL builds the PostgreSQL connection string from config.
func buildDatabaseURL(cfg *config.Config) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)
}

// executeCommand executes the specified migration command.
func executeCommand(m *migrate.Migrate, command string) error {
	switch command {
	case "up":
		return handleUp(m)
	case "down":
		return handleDown(m)
	case "step":
		return handleStep(m)
	case "version":
		return handleVersion(m)
	case "force":
		return handleForce(m)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
		return nil
	}
}

// handleUp applies all pending migrations.
func handleUp(m *migrate.Migrate) error {
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	log.Println("Migrations applied successfully")
	return nil
}

// handleDown rolls back all migrations.
func handleDown(m *migrate.Migrate) error {
	if err := m.Down(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to rollback migrations: %w", err)
	}
	log.Println("Migrations rolled back successfully")
	return nil
}

// handleStep applies the specified number of migration steps.
func handleStep(m *migrate.Migrate) error {
	if len(os.Args) < minArgsForStepOrForce {
		return fmt.Errorf("usage: migrate step <n>")
	}
	steps, err := strconv.Atoi(os.Args[2])
	if err != nil {
		return fmt.Errorf("invalid step value: %w", err)
	}
	if err := m.Steps(steps); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migration steps: %w", err)
	}
	log.Printf("Migration steps (%d) applied successfully\n", steps)
	return nil
}

// handleVersion shows the current migration version.
func handleVersion(m *migrate.Migrate) error {
	version, dirty, err := m.Version()
	if err != nil {
		return fmt.Errorf("failed to get migration version: %w", err)
	}
	fmt.Printf("Current version: %d\n", version)
	if dirty {
		fmt.Println("WARNING: Database is in dirty state")
	}
	return nil
}

// handleForce forces the database to a specific version.
func handleForce(m *migrate.Migrate) error {
	if len(os.Args) < minArgsForStepOrForce {
		return fmt.Errorf("usage: migrate force <version>")
	}
	version, err := strconv.Atoi(os.Args[2])
	if err != nil {
		return fmt.Errorf("invalid version value: %w", err)
	}
	if err := m.Force(version); err != nil {
		return fmt.Errorf("failed to force version: %w", err)
	}
	log.Printf("Forced database version to %d\n", version)
	return nil
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
	fmt.Println("\n  See .env.example for required environment variables.")
}
