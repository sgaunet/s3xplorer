// Package main is the entry point for the s3xplorer application, a web interface to browse S3 buckets
// It handles connection to S3, configuration loading, and web server management
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sgaunet/s3xplorer/pkg/app"
	configapp "github.com/sgaunet/s3xplorer/pkg/config"
)

// Package-level error variables.
var (
	// errNoAwsConfigMethod is returned when no method is available to initialize the AWS configuration.
	errNoAwsConfigMethod = errors.New("no method to initialize aws.Config")
)

func main() {
	var err error
	var fileName string
	var cfg configapp.Config
	flag.StringVar(&fileName, "f", "", "Configuration file")
	flag.Parse()

	// Check if the configuration file is provided
	if fileName == "" {
		fmt.Fprintf(os.Stderr, "Configuration file not provided. Exit 1")
		fmt.Fprintf(os.Stderr, "\nUsage:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Read the configuration file
	if cfg, err = configapp.ReadYamlCnxFile(fileName); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading configuration file: %s\n", err.Error())
		os.Exit(1)
	}
	// Initialize the logger
	l := initTrace(cfg.LogLevel)

	// Handle SIGTERM/SIGINT
	ctx, cancelFunc := context.WithCancel(context.Background())
	SetupCloseHandler(ctx, cancelFunc, l)

	// initialize the S3 client
	s3Client, err := initS3Client(ctx, cfg)
	if err != nil {
		l.Error("error initializing the S3 client: %s", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Create the app
	s := app.NewApp(cfg, s3Client)
	s.SetLogger(l)

	<-ctx.Done()
	l.Info("stop the server")
	if err := s.StopServer(); err != nil {
		l.Error("error stopping server", slog.String("error", err.Error()))
	}
}

// SetupCloseHandler handles graceful shutdown on SIGTERM/SIGINT signals.
func SetupCloseHandler(_ context.Context, cancelFunc context.CancelFunc, log *slog.Logger) {
	// Define the buffer size for the signal channel
	const signalChanBufferSize = 5
	c := make(chan os.Signal, signalChanBufferSize)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-c
		log.Info("INFO: signal received", slog.String("signal", s.String()))
		cancelFunc()
	}()
}

// initTrace initializes the logger.
func initTrace(debugLevel string) *slog.Logger {
	handlerOptions := &slog.HandlerOptions{
		Level: slog.LevelDebug,
		// AddSource: true,
	}

	switch debugLevel {
	case "debug":
		handlerOptions.Level = slog.LevelDebug
		handlerOptions.AddSource = true
	case "info":
		handlerOptions.Level = slog.LevelInfo
	case "warn":
		handlerOptions.Level = slog.LevelWarn
	case "error":
		handlerOptions.Level = slog.LevelError
	default:
		handlerOptions.Level = slog.LevelInfo
	}

	handler := slog.NewTextHandler(os.Stdout, handlerOptions)
	// handler := slog.NewJSONHandler(os.Stdout, nil) // JSON format
	logger := slog.New(handler)
	return logger
}

// initS3Client initializes the S3 client.
func initS3Client(ctx context.Context, configApp configapp.Config) (*s3.Client, error) {
	var cfg aws.Config
	cfg, err := GetAwsConfig(ctx, configApp)
	if err != nil {
		return nil, fmt.Errorf("error getting AWS config: %w", err)
	}

	// Apply additional S3-specific options if using a custom endpoint
	if configApp.S3endpoint != "" {
		// Use functional options pattern to configure the S3 client
		return s3.NewFromConfig(cfg, func(o *s3.Options) {
			// Set the custom endpoint URL
			o.BaseEndpoint = aws.String(configApp.S3endpoint)
			// Use path-style addressing (bucket name in the URL path)
			o.UsePathStyle = true
		}), nil
	}

	// Standard AWS S3 client configuration
	return s3.NewFromConfig(cfg), nil
}

// GetAwsConfig returns an aws.Config based on the provided configuration.
func GetAwsConfig(ctx context.Context, cfgApp configapp.Config) (aws.Config, error) {
	// Initialize an empty config
	var cfg aws.Config

	if cfgApp.S3endpoint != "" {
		// Parse the endpoint URL for validation
		_, err := url.Parse(cfgApp.S3endpoint)
		if err != nil {
			return aws.Config{}, fmt.Errorf("invalid S3 endpoint URL: %w", err)
		}

		// Load basic configuration with region & credentials
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(cfgApp.S3Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfgApp.S3accessKey,
				cfgApp.S3ApikKey,
				"",
			)),
		)
		if err != nil {
			return aws.Config{}, fmt.Errorf("error loading AWS config: %w", err)
		}

		// When we create the S3 client from this config, we'll modify it with custom endpoint
		// This is handled in the NewApp > initS3Client function, which calls:
		// s3.NewFromConfig(cfg) which gets this config
		// The s3.NewFromConfig will apply the custom endpoint when creating the client
		
		// Note: We're intentionally not using the deprecated endpoint resolvers here
		// When we create the S3 client, we'll use:
		// s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		//   o.BaseEndpoint = aws.String(cfgApp.S3endpoint)
		//   o.UsePathStyle = true
		// })
		// This happens in the initS3Client function

		return cfg, nil
	}

	if cfgApp.SsoAwsProfile != "" {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(cfgApp.SsoAwsProfile))
		if err != nil {
			// s.log.Error("Error loading SSO profile", slog.String("error", err.Error()))
			return cfg, fmt.Errorf("error loading SSO profile: %w", err)
		}
		// s.log.Debug("SSO profile loaded")
		return cfg, nil
	}

	if cfgApp.S3accessKey != "" && cfgApp.S3ApikKey != "" {
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return cfg, fmt.Errorf("error loading default config: %w", err)
		}
		// s.log.Debug("Default config loaded")
		return cfg, nil
	}
	return cfg, errNoAwsConfigMethod
}
