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
	// Create a temporary test file with valid YAML
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "valid_config.yaml")

	validYaml := `
s3endpoint: https://s3.example.com
accesskey: test-access-key
apikey: test-api-key
s3region: us-west-2
ssoawsprofile: test-profile
bucket: test-bucket
prefix: test-prefix
loglevel: debug
restoredays: 5
enableglacierrestore: true
`
	err := os.WriteFile(tmpFile, []byte(validYaml), 0644)
	require.NoError(t, err, "Failed to create test file")

	// Test reading the file
	cfg, err := config.ReadYamlCnxFile(tmpFile)
	require.NoError(t, err, "ReadYamlCnxFile should not return an error for valid YAML")

	// Verify all fields are correctly unmarshaled
	assert.Equal(t, "https://s3.example.com", cfg.S3endpoint)
	assert.Equal(t, "test-access-key", cfg.S3accessKey)
	assert.Equal(t, "test-api-key", cfg.S3ApikKey)
	assert.Equal(t, "us-west-2", cfg.S3Region)
	assert.Equal(t, "test-profile", cfg.SsoAwsProfile)
	assert.Equal(t, "test-bucket", cfg.Bucket)
	assert.Equal(t, "test-prefix", cfg.Prefix)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, 5, cfg.RestoreDays)
	assert.Equal(t, true, cfg.EnableGlacierRestore)
}

func TestReadYamlCnxFile_InvalidYaml(t *testing.T) {
	// Create a temporary test file with invalid YAML
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "invalid_config.yaml")

	invalidYaml := `
s3endpoint: https://s3.example.com
accesskey: test-access-key
apikey: test-api-key
s3region: us-west-2
ssoawsprofile: test-profile
bucket: test-bucket
prefix: test-prefix
loglevel: debug
restoredays: not-a-number  # Invalid value for int field
enableglacierrestore: true
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
	
	// Verify default values (all should be zero values)
	assert.Equal(t, "", cfg.S3endpoint)
	assert.Equal(t, "", cfg.S3accessKey)
	assert.Equal(t, "", cfg.S3ApikKey)
	assert.Equal(t, "", cfg.S3Region)
	assert.Equal(t, "", cfg.SsoAwsProfile)
	assert.Equal(t, "", cfg.Bucket)
	assert.Equal(t, "", cfg.Prefix)
	assert.Equal(t, "", cfg.LogLevel)
	assert.Equal(t, 0, cfg.RestoreDays)
	assert.Equal(t, false, cfg.EnableGlacierRestore)
}

func TestReadYamlCnxFile_PartialConfig(t *testing.T) {
	// Create a temporary test file with partial config
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "partial_config.yaml")

	partialYaml := `
s3endpoint: https://s3.example.com
bucket: test-bucket
restoredays: 7
`
	err := os.WriteFile(tmpFile, []byte(partialYaml), 0644)
	require.NoError(t, err, "Failed to create test file")

	// Test reading the file
	cfg, err := config.ReadYamlCnxFile(tmpFile)
	require.NoError(t, err, "ReadYamlCnxFile should not return an error for partial config")

	// Verify specified fields are set and others have zero values
	assert.Equal(t, "https://s3.example.com", cfg.S3endpoint)
	assert.Equal(t, "", cfg.S3accessKey)
	assert.Equal(t, "", cfg.S3ApikKey)
	assert.Equal(t, "", cfg.S3Region)
	assert.Equal(t, "", cfg.SsoAwsProfile)
	assert.Equal(t, "test-bucket", cfg.Bucket)
	assert.Equal(t, "", cfg.Prefix)
	assert.Equal(t, "", cfg.LogLevel)
	assert.Equal(t, 7, cfg.RestoreDays)
	assert.Equal(t, false, cfg.EnableGlacierRestore)
}
