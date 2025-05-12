// Package config provides configuration management for the s3xplorer application
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

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
}

// ReadYamlCnxFile reads a yaml file and returns a Config struct.
func ReadYamlCnxFile(filename string) (Config, error) {
	var config Config

	yamlFile, err := os.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading YAML file: %s\n", err)
		return config, fmt.Errorf("error reading config file %s: %w", filename, err)
	}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
		return config, fmt.Errorf("error parsing config YAML: %w", err)
	}
	return config, nil
}
