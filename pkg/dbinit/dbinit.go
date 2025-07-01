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

// InitializeDatabase initializes the database using embedded migrations
func InitializeDatabase(databaseURL string, logger *slog.Logger) (*sql.DB, error) {
	// Parse and validate database URL
	parsedURL, err := url.Parse(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid database URL: %w", err)
	}

	logger.Info("Initializing database", slog.String("host", parsedURL.Host))

	// Create dbmate instance
	db := dbmate.New(parsedURL)
	db.AutoDumpSchema = false
	db.MigrationsDir = []string{"."}
	db.SchemaFile = "schema.sql"

	// Set up embedded filesystem for migrations
	migrationFS, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to create migration filesystem: %w", err)
	}
	db.FS = migrationFS

	// Log available migrations
	migrationFiles, err := fs.ReadDir(migrations, "migrations")
	if err != nil {
		return nil, fmt.Errorf("failed to read migration files: %w", err)
	}

	logger.Info("Found migrations", slog.Int("count", len(migrationFiles)))
	for _, file := range migrationFiles {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".sql" {
			logger.Debug("Migration file", slog.String("name", file.Name()))
		}
	}

	// Create database if it doesn't exist
	logger.Info("Creating database if not exists")
	if err := db.CreateAndMigrate(); err != nil {
		return nil, fmt.Errorf("failed to create and migrate database: %w", err)
	}

	logger.Info("Database initialization completed successfully")

	// Open direct database connection
	sqlDB, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	if err := sqlDB.PingContext(context.Background()); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("Database connection established successfully")
	return sqlDB, nil
}

// MigrateDatabase runs pending migrations on an existing database
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
	if err := db.Migrate(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info("Database migrations completed successfully")
	return nil
}