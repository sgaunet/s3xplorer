package dbinit

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEmbeddedMigrations verifies that all migration files are properly embedded
func TestEmbeddedMigrations(t *testing.T) {
	// Check that the migrations filesystem is accessible
	migrationFiles, err := fs.ReadDir(migrations, "migrations")
	require.NoError(t, err, "Should be able to read embedded migrations directory")

	// Count SQL migration files
	sqlFiles := 0
	var foundFiles []string
	for _, file := range migrationFiles {
		if !file.IsDir() && len(file.Name()) > 4 && file.Name()[len(file.Name())-4:] == ".sql" {
			sqlFiles++
			foundFiles = append(foundFiles, file.Name())
		}
	}

	// We should have exactly 3 migration files
	assert.Equal(t, 3, sqlFiles, "Should have exactly 3 SQL migration files embedded")

	// Check for specific expected migrations
	expectedMigrations := []string{
		"20250629000001_create_s3_objects.sql",
		"20250702000001_add_marked_for_deletion.sql", 
		"20250702000002_add_deletion_tracking_to_scan_jobs.sql",
	}

	for _, expected := range expectedMigrations {
		assert.Contains(t, foundFiles, expected, "Should contain migration: %s", expected)
	}

	t.Logf("Found embedded migration files: %v", foundFiles)
}

// TestMigrationContent verifies that migration files contain expected content
func TestMigrationContent(t *testing.T) {
	// Test that the marked_for_deletion migration contains the expected ALTER TABLE statement
	migrationContent, err := fs.ReadFile(migrations, "migrations/20250702000001_add_marked_for_deletion.sql")
	require.NoError(t, err, "Should be able to read marked_for_deletion migration")
	
	content := string(migrationContent)
	assert.Contains(t, content, "ALTER TABLE s3_objects ADD COLUMN marked_for_deletion BOOLEAN DEFAULT FALSE", 
		"Migration should contain marked_for_deletion column addition")
	assert.Contains(t, content, "CREATE INDEX idx_s3_objects_marked_for_deletion", 
		"Migration should contain index creation for marked_for_deletion")

	// Test that the scan jobs tracking migration contains expected content
	trackingContent, err := fs.ReadFile(migrations, "migrations/20250702000002_add_deletion_tracking_to_scan_jobs.sql")
	require.NoError(t, err, "Should be able to read scan jobs tracking migration")
	
	trackingStr := string(trackingContent)
	assert.Contains(t, trackingStr, "ALTER TABLE scan_jobs ADD COLUMN objects_deleted INTEGER DEFAULT 0",
		"Migration should contain objects_deleted column addition")
	assert.Contains(t, trackingStr, "ALTER TABLE scan_jobs ADD COLUMN objects_updated INTEGER DEFAULT 0",
		"Migration should contain objects_updated column addition") 
	assert.Contains(t, trackingStr, "ALTER TABLE scan_jobs ADD COLUMN objects_created INTEGER DEFAULT 0",
		"Migration should contain objects_created column addition")
}