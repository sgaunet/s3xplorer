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

// S3Config contains S3-related configuration
type S3Config struct {
	Endpoint         string `yaml:"endpoint"`
	AccessKey        string `yaml:"access_key"`
	ApiKey           string `yaml:"api_key"`
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

// DatabaseConfig contains database-related configuration
type DatabaseConfig struct {
	URL string `yaml:"url"`
}

// ScanConfig contains scanning-related configuration
type ScanConfig struct {
	EnableBackgroundScan bool   `yaml:"enable_background_scan"`
	CronSchedule         string `yaml:"cron_schedule"`
	EnableInitialScan    bool   `yaml:"enable_initial_scan"`
	EnableDeletionSync   bool   `yaml:"enable_deletion_sync"`
}

// BucketSyncConfig contains bucket synchronization configuration
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
	
	// Legacy flat fields for backward compatibility
	// These will be deprecated in future versions
	S3endpoint         string `yaml:"s3endpoint,omitempty"`
	S3accessKey        string `yaml:"accesskey,omitempty"`
	S3ApikKey          string `yaml:"apikey,omitempty"`
	S3Region           string `yaml:"s3region,omitempty"`
	SsoAwsProfile      string `yaml:"ssoawsprofile,omitempty"`
	Bucket             string `yaml:"bucket,omitempty"`
	Prefix             string `yaml:"prefix,omitempty"`
	RestoreDays        int    `yaml:"restoredays,omitempty"`
	EnableGlacierRestore bool  `yaml:"enableglacierrestore,omitempty"`
	DatabaseURL        string `yaml:"database_url,omitempty"`
	ScanCronSchedule   string `yaml:"scan_cron_schedule,omitempty"`
	EnableBackgroundScan bool `yaml:"enable_background_scan,omitempty"`
	EnableInitialScan  bool   `yaml:"enable_initial_scan,omitempty"`
	EnableDeletionSync bool   `yaml:"enable_deletion_sync,omitempty"`
	EnableBucketSync      bool   `yaml:"enable_bucket_sync,omitempty"`
	BucketSyncThreshold   string `yaml:"bucket_sync_threshold,omitempty"`
	BucketDeleteThreshold string `yaml:"bucket_delete_threshold,omitempty"`
	BucketMaxRetries      int    `yaml:"bucket_max_retries,omitempty"`
	SkipBucketValidation  bool   `yaml:"skip_bucket_validation,omitempty"`
	LegacyLogLevel        string `yaml:"loglevel,omitempty"`
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
	
	// Migrate legacy flat fields to new hierarchical structure
	config.migrateFromLegacy()
	
	// Set BucketLocked flag if bucket is explicitly specified in config
	config.S3.BucketLocked = config.S3.Bucket != ""
	
	// Set default values
	config.setDefaults()

	return config, nil
}

// migrateFromLegacy migrates legacy flat configuration fields to the new hierarchical structure
func (c *Config) migrateFromLegacy() {
	// Migrate S3 configuration
	if c.S3endpoint != "" && c.S3.Endpoint == "" {
		c.S3.Endpoint = c.S3endpoint
	}
	if c.S3accessKey != "" && c.S3.AccessKey == "" {
		c.S3.AccessKey = c.S3accessKey
	}
	if c.S3ApikKey != "" && c.S3.ApiKey == "" {
		c.S3.ApiKey = c.S3ApikKey
	}
	if c.S3Region != "" && c.S3.Region == "" {
		c.S3.Region = c.S3Region
	}
	if c.SsoAwsProfile != "" && c.S3.SsoAwsProfile == "" {
		c.S3.SsoAwsProfile = c.SsoAwsProfile
	}
	if c.Bucket != "" && c.S3.Bucket == "" {
		c.S3.Bucket = c.Bucket
	}
	if c.Prefix != "" && c.S3.Prefix == "" {
		c.S3.Prefix = c.Prefix
	}
	if c.RestoreDays > 0 && c.S3.RestoreDays == 0 {
		c.S3.RestoreDays = c.RestoreDays
	}
	if c.EnableGlacierRestore && !c.S3.EnableGlacierRestore {
		c.S3.EnableGlacierRestore = c.EnableGlacierRestore
	}
	if c.SkipBucketValidation && !c.S3.SkipBucketValidation {
		c.S3.SkipBucketValidation = c.SkipBucketValidation
	}
	
	// Migrate Database configuration
	if c.DatabaseURL != "" && c.Database.URL == "" {
		c.Database.URL = c.DatabaseURL
	}
	
	// Migrate Scan configuration
	if c.EnableBackgroundScan && !c.Scan.EnableBackgroundScan {
		c.Scan.EnableBackgroundScan = c.EnableBackgroundScan
	}
	if c.ScanCronSchedule != "" && c.Scan.CronSchedule == "" {
		c.Scan.CronSchedule = c.ScanCronSchedule
	}
	if c.EnableInitialScan && !c.Scan.EnableInitialScan {
		c.Scan.EnableInitialScan = c.EnableInitialScan
	}
	if c.EnableDeletionSync && !c.Scan.EnableDeletionSync {
		c.Scan.EnableDeletionSync = c.EnableDeletionSync
	}
	
	// Migrate BucketSync configuration
	if c.EnableBucketSync && !c.BucketSync.Enable {
		c.BucketSync.Enable = c.EnableBucketSync
	}
	if c.BucketSyncThreshold != "" && c.BucketSync.SyncThreshold == "" {
		c.BucketSync.SyncThreshold = c.BucketSyncThreshold
	}
	if c.BucketDeleteThreshold != "" && c.BucketSync.DeleteThreshold == "" {
		c.BucketSync.DeleteThreshold = c.BucketDeleteThreshold
	}
	if c.BucketMaxRetries > 0 && c.BucketSync.MaxRetries == 0 {
		c.BucketSync.MaxRetries = c.BucketMaxRetries
	}
	
	// Migrate LogLevel
	if c.LegacyLogLevel != "" && c.LogLevel == "" {
		c.LogLevel = c.LegacyLogLevel
	}
}

// setDefaults sets default values for configuration fields
func (c *Config) setDefaults() {
	// Set default scan cron schedule
	if c.Scan.CronSchedule == "" {
		c.Scan.CronSchedule = "0 0 2 * * *" // Daily at 2 AM (with seconds field)
	}
	
	// Set default database URL
	if c.Database.URL == "" {
		c.Database.URL = "postgres://postgres:postgres@localhost:5432/s3xplorer?sslmode=disable"
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
