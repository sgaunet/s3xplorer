// Package config provides configuration management for the s3xplorer application
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// ErrIsDirectory is returned when a file operation is performed on a directory.
var ErrIsDirectory = errors.New("expected file but got directory")

// Config is the struct for the configuration.
type Config struct {
	S3endpoint         string `yaml:"s3endpoint"`
	S3accessKey        string `yaml:"accesskey"`
	S3ApikKey          string `yaml:"apikey"`
	S3Region           string `yaml:"s3region"`
	SsoAwsProfile      string `yaml:"ssoawsprofile"`
	Bucket             string `yaml:"bucket"`
	Prefix             string `yaml:"prefix"`
	LogLevel           string `yaml:"loglevel"`
	RestoreDays        int    `yaml:"restoredays"`
	EnableGlacierRestore bool  `yaml:"enableglacierrestore"`
	// Database configuration
	DatabaseURL        string `yaml:"database_url"`
	// Background job configuration
	ScanCronSchedule   string `yaml:"scan_cron_schedule"`
	EnableBackgroundScan bool `yaml:"enable_background_scan"`
	// Initial scan configuration
	EnableInitialScan  bool   `yaml:"enable_initial_scan"`
	// Deletion sync configuration
	EnableDeletionSync bool   `yaml:"enable_deletion_sync"`
	// Bucket sync configuration
	EnableBucketSync      bool   `yaml:"enable_bucket_sync"`
	BucketSyncThreshold   string `yaml:"bucket_sync_threshold"`
	BucketDeleteThreshold string `yaml:"bucket_delete_threshold"`
	BucketMaxRetries      int    `yaml:"bucket_max_retries"`
	// Skip bucket validation (HeadBucket operation)
	SkipBucketValidation  bool   `yaml:"skip_bucket_validation"`
	// Not serialized, but used to track whether bucket was explicitly set in config
	BucketLocked       bool   `yaml:"-"`
}

// ReadYamlCnxFile reads a yaml file and returns a Config struct.
func ReadYamlCnxFile(filename string) (Config, error) {
	var config Config

	// Sanitize the path to prevent path traversal attacks
	cleanPath := filepath.Clean(filename)
	// Additional safety check - ensure the file exists and is a regular file
	fileInfo, err := os.Stat(cleanPath)
	if err != nil {
		return config, fmt.Errorf("error accessing config file %s: %w", filename, err)
	}

	if fileInfo.IsDir() {
		return config, fmt.Errorf("%w: %s", ErrIsDirectory, filename)
	}

	yamlFile, err := os.ReadFile(cleanPath)
	if err != nil {
		fmt.Printf("Error reading YAML file: %s\n", err)
		return config, fmt.Errorf("error reading config file %s: %w", filename, err)
	}
	
	// Parse YAML into config structure
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		return config, fmt.Errorf("error parsing YAML from %s: %w", filename, err)
	}
	
	// Set BucketLocked flag if bucket is explicitly specified in config
	config.BucketLocked = config.Bucket != ""
	
	// Set default values for new fields
	if config.ScanCronSchedule == "" {
		config.ScanCronSchedule = "0 0 2 * * *" // Daily at 2 AM (with seconds field)
	}
	if config.DatabaseURL == "" {
		config.DatabaseURL = "postgres://postgres:postgres@localhost:5432/s3xplorer?sslmode=disable"
	}
	// Set default values for bucket sync configuration
	if config.BucketSyncThreshold == "" {
		config.BucketSyncThreshold = "24h" // Mark as inaccessible after 24 hours
	}
	if config.BucketDeleteThreshold == "" {
		config.BucketDeleteThreshold = "168h" // Delete after 7 days (168 hours)
	}
	if config.BucketMaxRetries == 0 {
		config.BucketMaxRetries = 3 // Default to 3 retries for bucket access checks
	}
	// EnableInitialScan defaults to false for safety
	// EnableDeletionSync defaults to true since it's generally desired behavior
	// EnableBucketSync defaults to true to enable bucket-level synchronization
	// Note: for existing configs without this field, YAML unmarshaling will leave it as false (zero value)
	// So we need to set it explicitly if not specified in the config
	// For safety, we'll default to true to enable the new functionality by default

	return config, nil
}
