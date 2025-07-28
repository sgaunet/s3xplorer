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
	"strings"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	_ "github.com/lib/pq"
	"github.com/sgaunet/s3xplorer/pkg/app"
	configapp "github.com/sgaunet/s3xplorer/pkg/config"
	"github.com/sgaunet/s3xplorer/pkg/dbinit"
	"github.com/sgaunet/s3xplorer/pkg/dbsvc"
	"github.com/sgaunet/s3xplorer/pkg/scanner"
	"github.com/sgaunet/s3xplorer/pkg/scheduler"
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

	// Initialize database with embedded migrations
	dbConn, err := dbinit.InitializeDatabase(cfg.DatabaseURL, l)
	if err != nil {
		l.Error("error initializing database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer dbConn.Close()

	// Create services
	dbService := dbsvc.NewService(cfg, dbConn)
	dbService.SetLogger(l)

	scannerService := scanner.NewService(cfg, s3Client, dbConn)
	scannerService.SetLogger(l)

	// Create scheduler
	scheduler := scheduler.NewScheduler(cfg, dbConn, scannerService)
	scheduler.SetLogger(l)

	// Perform initial scan if enabled
	if cfg.EnableInitialScan {
		l.Info("Performing initial bucket scan")
		if err := scannerService.DiscoverAndScanAllBuckets(ctx); err != nil {
			l.Error("error during initial scan", slog.String("error", err.Error()))
			// Don't exit - continue with application startup even if initial scan fails
		} else {
			l.Info("Initial bucket scan completed successfully")
		}
	}

	// Start scheduler
	if err := scheduler.Start(ctx); err != nil {
		l.Error("error starting scheduler", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Create the app
	s := app.NewApp(cfg, s3Client, dbService)
	s.SetLogger(l)

	<-ctx.Done()
	l.Info("stop the server")
	scheduler.Stop()
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
		// Check if this is an AWS S3 endpoint (contains amazonaws.com)
		isAwsEndpoint := strings.Contains(configApp.S3endpoint, "amazonaws.com")
		usePathStyle := !isAwsEndpoint

		// fmt.Printf("Custom endpoint detected - AWS: %t, UsePathStyle: %t\n", isAwsEndpoint, usePathStyle)

		// Use functional options pattern to configure the S3 client
		return s3.NewFromConfig(cfg, func(o *s3.Options) {
			// Set the custom endpoint URL
			o.BaseEndpoint = aws.String(configApp.S3endpoint)
			// Use path-style addressing only for non-AWS endpoints (like MinIO)
			// AWS S3 should use virtual-hosted-style (UsePathStyle = false)
			o.UsePathStyle = usePathStyle
			// Ensure region is set correctly for both AWS and custom endpoints
			o.Region = configApp.S3Region
		}), nil
	}

	// Standard AWS S3 client configuration
	// For AWS S3, we need to ensure the region is properly set
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.Region = configApp.S3Region
	}), nil
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
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(cfgApp.S3Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfgApp.S3accessKey,
				cfgApp.S3ApikKey,
				"",
			)),
		)
		if err != nil {
			return cfg, fmt.Errorf("error loading default config: %w", err)
		}
		// s.log.Debug("Default config loaded with static credentials")
		return cfg, nil
	}

	// Fall back to default credential chain (includes EC2 IAM role)
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(cfgApp.S3Region),
	)
	if err != nil {
		return cfg, fmt.Errorf("error loading default config: %w", err)
	}
	// This will use the default credential chain:
	// 1. Environment variables
	// 2. Shared credentials file
	// 3. EC2 IAM role
	// 4. ECS task role
	// 5. etc.
	return cfg, nil
}
