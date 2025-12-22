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

// S3Config contains S3-related configuration.
type S3Config struct {
	Endpoint         string `yaml:"endpoint"`
	AccessKey        string `yaml:"access_key"`
	APIKey           string `yaml:"api_key"`
	Region           string `yaml:"region"`
	SsoAwsProfile    string `yaml:"sso_aws_profile"`
	Bucket           string `yaml:"bucket"`
	Prefix           string `yaml:"prefix"`
	RestoreDays      int    `yaml:"restore_days"`
	EnableGlacierRestore bool `yaml:"enable_glacier_restore"`
	SkipBucketValidation bool `yaml:"skip_bucket_validation"`
	// Not serialized, but used to track whether bucket was explicitly set in config
	BucketLocked     bool   `yaml:"-"`
}

// DatabaseConfig contains database-related configuration.
type DatabaseConfig struct {
	URL              string `yaml:"url"`
	MaxOpenConns     int    `yaml:"max_open_conns"`
	MaxIdleConns     int    `yaml:"max_idle_conns"`
	ConnMaxLifetime  string `yaml:"conn_max_lifetime"`
	ConnMaxIdleTime  string `yaml:"conn_max_idle_time"`
}

// ScanConfig contains scanning-related configuration.
type ScanConfig struct {
	EnableBackgroundScan bool   `yaml:"enable_background_scan"`
	CronSchedule         string `yaml:"cron_schedule"`
	EnableInitialScan    bool   `yaml:"enable_initial_scan"`
	EnableDeletionSync   bool   `yaml:"enable_deletion_sync"`
}

// BucketSyncConfig contains bucket synchronization configuration.
type BucketSyncConfig struct {
	Enable          bool   `yaml:"enable"`
	SyncThreshold   string `yaml:"sync_threshold"`
	DeleteThreshold string `yaml:"delete_threshold"`
	MaxRetries      int    `yaml:"max_retries"`
}

// Config is the struct for the configuration.
type Config struct {
	S3         S3Config         `yaml:"s3"`
	Database   DatabaseConfig   `yaml:"database"`
	Scan       ScanConfig       `yaml:"scan"`
	BucketSync BucketSyncConfig `yaml:"bucket_sync"`
	LogLevel   string           `yaml:"log_level"`
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
	config.S3.BucketLocked = config.S3.Bucket != ""
	
	// Set default values
	config.setDefaults()

	return config, nil
}

// setDefaults sets default values for configuration fields.
func (c *Config) setDefaults() {
	// Set default scan cron schedule
	if c.Scan.CronSchedule == "" {
		c.Scan.CronSchedule = "0 0 2 * * *" // Daily at 2 AM (with seconds field)
	}
	
	// Set default database URL
	if c.Database.URL == "" {
		c.Database.URL = "postgres://postgres:postgres@localhost:5432/s3xplorer?sslmode=disable"
	}

	// Set default database pool settings
	if c.Database.MaxOpenConns == 0 {
		c.Database.MaxOpenConns = 25
	}
	if c.Database.MaxIdleConns == 0 {
		c.Database.MaxIdleConns = 5
	}
	if c.Database.ConnMaxLifetime == "" {
		c.Database.ConnMaxLifetime = "5m"
	}
	if c.Database.ConnMaxIdleTime == "" {
		c.Database.ConnMaxIdleTime = "1m"
	}

	// Set default bucket sync configuration
	if c.BucketSync.SyncThreshold == "" {
		c.BucketSync.SyncThreshold = "24h" // Mark as inaccessible after 24 hours
	}
	if c.BucketSync.DeleteThreshold == "" {
		c.BucketSync.DeleteThreshold = "168h" // Delete after 7 days (168 hours)
	}
	if c.BucketSync.MaxRetries == 0 {
		c.BucketSync.MaxRetries = 3 // Default to 3 retries for bucket access checks
	}
}
