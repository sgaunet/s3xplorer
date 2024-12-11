package config

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Config is the struct for the configuration
type Config struct {
	S3endpoint    string `yaml:"s3endpoint"`
	S3accessKey   string `yaml:"accesskey"`
	S3ApikKey     string `yaml:"apikey"`
	S3Region      string `yaml:"s3region"`
	SsoAwsProfile string `yaml:"ssoawsprofile"`
	Bucket        string `yaml:"bucket"`
	Prefix        string `yaml:"prefix"`
	LogLevel      string `yaml:"loglevel"`
}

// ReadYamlCnxFile reads a yaml file and returns a Config struct
func ReadYamlCnxFile(filename string) (Config, error) {
	var config Config

	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading YAML file: %s\n", err)
		return config, err
	}

	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		fmt.Printf("Error parsing YAML file: %s\n", err)
		return config, err
	}
	return config, err
}
