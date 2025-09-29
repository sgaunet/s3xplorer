package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sgaunet/s3xplorer/pkg/config"
)

func TestReadYamlCnxFile_ValidFile(t *testing.T) {
	// Create a temporary test file with valid hierarchical YAML
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "valid_config.yaml")

	validYaml := `
s3:
  endpoint: https://s3.example.com
  access_key: test-access-key
  api_key: test-api-key
  region: us-west-2
  sso_aws_profile: test-profile
  bucket: test-bucket
  prefix: test-prefix
  restore_days: 5
  enable_glacier_restore: true
database:
  url: postgres://custom@localhost:5432/mydb
scan:
  enable_background_scan: true
  cron_schedule: "0 */6 * * *"
bucket_sync:
  enable: true
log_level: debug
`
	err := os.WriteFile(tmpFile, []byte(validYaml), 0644)
	require.NoError(t, err, "Failed to create test file")

	// Test reading the file
	cfg, err := config.ReadYamlCnxFile(tmpFile)
	require.NoError(t, err, "ReadYamlCnxFile should not return an error for valid YAML")

	// Verify all fields are correctly unmarshaled
	assert.Equal(t, "https://s3.example.com", cfg.S3.Endpoint)
	assert.Equal(t, "test-access-key", cfg.S3.AccessKey)
	assert.Equal(t, "test-api-key", cfg.S3.APIKey)
	assert.Equal(t, "us-west-2", cfg.S3.Region)
	assert.Equal(t, "test-profile", cfg.S3.SsoAwsProfile)
	assert.Equal(t, "test-bucket", cfg.S3.Bucket)
	assert.Equal(t, "test-prefix", cfg.S3.Prefix)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, 5, cfg.S3.RestoreDays)
	assert.Equal(t, true, cfg.S3.EnableGlacierRestore)
	assert.Equal(t, "postgres://custom@localhost:5432/mydb", cfg.Database.URL)
	assert.Equal(t, true, cfg.Scan.EnableBackgroundScan)
	assert.Equal(t, "0 */6 * * *", cfg.Scan.CronSchedule)
	assert.Equal(t, true, cfg.BucketSync.Enable)
}

func TestReadYamlCnxFile_InvalidYaml(t *testing.T) {
	// Create a temporary test file with invalid YAML
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid_config.yaml")

	invalidYaml := `
s3:
  endpoint: https://s3.example.com
  access_key: test-access-key
  api_key: test-api-key
  region: us-west-2
  bucket: test-bucket
  prefix: test-prefix
  restore_days: not-a-number  # Invalid value for int field
  enable_glacier_restore: true
log_level: debug
`
	err := os.WriteFile(tmpFile, []byte(invalidYaml), 0644)
	require.NoError(t, err, "Failed to create test file")

	// Test reading the file
	_, err = config.ReadYamlCnxFile(tmpFile)
	assert.Error(t, err, "ReadYamlCnxFile should return an error for invalid YAML")
}

func TestReadYamlCnxFile_NonExistentFile(t *testing.T) {
	// Test reading a non-existent file
	_, err := config.ReadYamlCnxFile("/path/to/non-existent/file.yaml")
	assert.Error(t, err, "ReadYamlCnxFile should return an error for non-existent file")
}

func TestReadYamlCnxFile_EmptyFile(t *testing.T) {
	// Create a temporary empty test file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty_config.yaml")

	err := os.WriteFile(tmpFile, []byte{}, 0644)
	require.NoError(t, err, "Failed to create empty test file")

	// Test reading the file
	cfg, err := config.ReadYamlCnxFile(tmpFile)
	assert.NoError(t, err, "ReadYamlCnxFile should not return an error for empty file")
	
	// Verify default values (all should be zero values except for defaults)
	assert.Equal(t, "", cfg.S3.Endpoint)
	assert.Equal(t, "", cfg.S3.AccessKey)
	assert.Equal(t, "", cfg.S3.APIKey)
	assert.Equal(t, "", cfg.S3.Region)
	assert.Equal(t, "", cfg.S3.SsoAwsProfile)
	assert.Equal(t, "", cfg.S3.Bucket)
	assert.Equal(t, "", cfg.S3.Prefix)
	assert.Equal(t, "", cfg.LogLevel)
	assert.Equal(t, 0, cfg.S3.RestoreDays)
	assert.Equal(t, false, cfg.S3.EnableGlacierRestore)
	// Check defaults
	assert.Equal(t, "postgres://postgres:postgres@localhost:5432/s3xplorer?sslmode=disable", cfg.Database.URL)
	assert.Equal(t, "0 0 2 * * *", cfg.Scan.CronSchedule)
	assert.Equal(t, "24h", cfg.BucketSync.SyncThreshold)
	assert.Equal(t, "168h", cfg.BucketSync.DeleteThreshold)
	assert.Equal(t, 3, cfg.BucketSync.MaxRetries)
}

func TestReadYamlCnxFile_PartialConfig(t *testing.T) {
	// Create a temporary test file with partial config
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "partial_config.yaml")

	partialYaml := `
s3:
  endpoint: https://s3.example.com
  bucket: test-bucket
  restore_days: 7
`
	err := os.WriteFile(tmpFile, []byte(partialYaml), 0644)
	require.NoError(t, err, "Failed to create test file")

	// Test reading the file
	cfg, err := config.ReadYamlCnxFile(tmpFile)
	require.NoError(t, err, "ReadYamlCnxFile should not return an error for partial config")

	// Verify specified fields are set and others have zero values
	assert.Equal(t, "https://s3.example.com", cfg.S3.Endpoint)
	assert.Equal(t, "", cfg.S3.AccessKey)
	assert.Equal(t, "", cfg.S3.APIKey)
	assert.Equal(t, "", cfg.S3.Region)
	assert.Equal(t, "", cfg.S3.SsoAwsProfile)
	assert.Equal(t, "test-bucket", cfg.S3.Bucket)
	assert.Equal(t, "", cfg.S3.Prefix)
	assert.Equal(t, "", cfg.LogLevel)
	assert.Equal(t, 7, cfg.S3.RestoreDays)
	assert.Equal(t, false, cfg.S3.EnableGlacierRestore)
}

func TestReadYamlCnxFile_NewHierarchicalFormat(t *testing.T) {
	// Create a temporary test file with new hierarchical format
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "hierarchical_config.yaml")

	hierarchicalYaml := `
s3:
  endpoint: https://s3.example.com
  access_key: test-access-key
  api_key: test-api-key
  region: us-west-2
  sso_aws_profile: test-profile
  bucket: test-bucket
  prefix: test-prefix
  restore_days: 5
  enable_glacier_restore: true
  skip_bucket_validation: true
database:
  url: postgres://custom@localhost:5432/mydb
scan:
  enable_background_scan: true
  cron_schedule: "0 */6 * * *"
  enable_initial_scan: true
  enable_deletion_sync: true
bucket_sync:
  enable: true
  sync_threshold: "12h"
  delete_threshold: "48h"
  max_retries: 5
log_level: debug
`
	err := os.WriteFile(tmpFile, []byte(hierarchicalYaml), 0644)
	require.NoError(t, err, "Failed to create test file")

	// Test reading the file
	cfg, err := config.ReadYamlCnxFile(tmpFile)
	require.NoError(t, err, "ReadYamlCnxFile should not return an error for hierarchical config")

	// Verify S3 config
	assert.Equal(t, "https://s3.example.com", cfg.S3.Endpoint)
	assert.Equal(t, "test-access-key", cfg.S3.AccessKey)
	assert.Equal(t, "test-api-key", cfg.S3.APIKey)
	assert.Equal(t, "us-west-2", cfg.S3.Region)
	assert.Equal(t, "test-profile", cfg.S3.SsoAwsProfile)
	assert.Equal(t, "test-bucket", cfg.S3.Bucket)
	assert.Equal(t, "test-prefix", cfg.S3.Prefix)
	assert.Equal(t, 5, cfg.S3.RestoreDays)
	assert.Equal(t, true, cfg.S3.EnableGlacierRestore)
	assert.Equal(t, true, cfg.S3.SkipBucketValidation)
	assert.Equal(t, true, cfg.S3.BucketLocked)
	
	// Verify Database config
	assert.Equal(t, "postgres://custom@localhost:5432/mydb", cfg.Database.URL)
	
	// Verify Scan config
	assert.Equal(t, true, cfg.Scan.EnableBackgroundScan)
	assert.Equal(t, "0 */6 * * *", cfg.Scan.CronSchedule)
	assert.Equal(t, true, cfg.Scan.EnableInitialScan)
	assert.Equal(t, true, cfg.Scan.EnableDeletionSync)
	
	// Verify BucketSync config
	assert.Equal(t, true, cfg.BucketSync.Enable)
	assert.Equal(t, "12h", cfg.BucketSync.SyncThreshold)
	assert.Equal(t, "48h", cfg.BucketSync.DeleteThreshold)
	assert.Equal(t, 5, cfg.BucketSync.MaxRetries)
	
	// Verify LogLevel
	assert.Equal(t, "debug", cfg.LogLevel)
}
