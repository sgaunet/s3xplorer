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

	return config, nil
}
