// Package dbinit provides database initialization and migration functionality.
package dbinit

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"net/url"
	"path/filepath"

	"github.com/amacneil/dbmate/v2/pkg/dbmate"
	_ "github.com/amacneil/dbmate/v2/pkg/driver/postgres" // PostgreSQL driver for dbmate
	_ "github.com/lib/pq"                                 // PostgreSQL driver
)

//go:embed migrations
var migrations embed.FS

// InitializeDatabase initializes the database using embedded migrations.
func InitializeDatabase(ctx context.Context, databaseURL string, logger *slog.Logger) (*sql.DB, error) {
	// Parse and validate database URL
	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid database URL: %w", err)
	}

	logger.Info("Initializing database", slog.String("host", parsedURL.Host))

	// Set up and run migrations
	err = setupAndRunMigrations(parsedURL, logger)
	if err != nil {
		return nil, err
	}

	logger.Info("Database initialization completed successfully")

	// Open and test database connection
	return openAndTestConnection(ctx, databaseURL, logger)
}

// setupAndRunMigrations configures dbmate and runs migrations.
func setupAndRunMigrations(parsedURL *url.URL, logger *slog.Logger) error {
	// Create dbmate instance
	db := dbmate.New(parsedURL)
	db.AutoDumpSchema = false
	db.MigrationsDir = []string{"."}
	db.SchemaFile = "schema.sql"

	// Set up embedded filesystem for migrations
	migrationFS, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration filesystem: %w", err)
	}
	db.FS = migrationFS

	// Log available migrations
	err = logMigrations(logger)
	if err != nil {
		return err
	}

	// Create database if it doesn't exist
	logger.Info("Creating database if not exists")
	err = db.CreateAndMigrate()
	if err != nil {
		return fmt.Errorf("failed to create and migrate database: %w", err)
	}

	return nil
}

// logMigrations logs the available migration files.
func logMigrations(logger *slog.Logger) error {
	migrationFiles, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("failed to read migration files: %w", err)
	}

	logger.Info("Found migrations", slog.Int("count", len(migrationFiles)))
	for _, file := range migrationFiles {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".sql" {
			logger.Debug("Migration file", slog.String("name", file.Name()))
		}
	}
	return nil
}

// openAndTestConnection opens a database connection and tests it.
func openAndTestConnection(ctx context.Context, databaseURL string, logger *slog.Logger) (*sql.DB, error) {
	sqlDB, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	err = sqlDB.PingContext(ctx)
	if err != nil {
		closeErr := sqlDB.Close()
		if closeErr != nil {
			logger.Error("Failed to close database connection", slog.String("error", closeErr.Error()))
		}
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection established successfully")
	return sqlDB, nil
}

// MigrateDatabase runs pending migrations on an existing database.
func MigrateDatabase(databaseURL string, logger *slog.Logger) error {
	// Parse database URL
	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		return fmt.Errorf("invalid database URL: %w", err)
	}

	logger.Info("Running database migrations", slog.String("host", parsedURL.Host))

	// Create dbmate instance
	db := dbmate.New(parsedURL)
	db.AutoDumpSchema = false
	db.MigrationsDir = []string{"migrations"}

	// Set up embedded filesystem for migrations
	migrationFS, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create migration filesystem: %w", err)
	}
	db.FS = migrationFS

	// Run migrations
	err = db.Migrate()
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info("Database migrations completed successfully")
	return nil
}