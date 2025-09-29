// Package main is the entry point for the s3xplorer application, a web interface to browse S3 buckets
// It handles connection to S3, configuration loading, and web server management
package main

import (
	"context"
	"database/sql"
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

//go:generate go tool github.com/sqlc-dev/sqlc/cmd/sqlc generate -f sqlc.yaml

// ErrConfigFileNotProvided is returned when no configuration file is provided.
var ErrConfigFileNotProvided = errors.New("configuration file not provided")

func main() {
	// Parse configuration
	cfg, err := parseConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// Initialize the logger
	l := initTrace(cfg.LogLevel)

	// Handle SIGTERM/SIGINT
	ctx, cancelFunc := context.WithCancel(context.Background())
	SetupCloseHandler(ctx, cancelFunc, l)

	// Initialize infrastructure
	s3Client, dbConn, err := initInfrastructure(ctx, cfg, l)
	var dbService *dbsvc.Service
	var scannerService *scanner.Service
	var scheduler *scheduler.Scheduler

	if err != nil {
		l.Error("Failed to initialize infrastructure", slog.String("error", err.Error()))
		l.Warn("Starting application in degraded mode without database connectivity")

		// Initialize services without database connection
		dbService = nil // Will be handled gracefully by the app
		// scannerService and scheduler remain nil when database is unavailable
	} else {
		defer func() {
			if err := dbConn.Close(); err != nil {
				l.Error("Failed to close database connection", slog.String("error", err.Error()))
			}
		}()

		// Initialize services
		dbService, scannerService, scheduler = initServices(cfg, s3Client, dbConn, l)
	}

	// Create and start the web server immediately (handles nil dbService gracefully)
	s := app.NewApp(cfg, s3Client, dbService)
	s.SetLogger(l)

	// Start background processes after web server is running
	if scannerService != nil && scheduler != nil {
		// Run initial scan in background to avoid blocking web server startup
		go func() {
			l.Info("Starting initial scan in background - web server is ready for health checks")
			performInitialScan(ctx, cfg, scannerService, l)

			// Start scheduler after initial scan completes
			if err := scheduler.Start(ctx); err != nil {
				l.Error("error starting scheduler", slog.String("error", err.Error()))
				// Continue without scheduler instead of crashing
			}
		}()
	}

	// Wait for shutdown signal
	<-ctx.Done()
	shutdown(s, scheduler, l)
}

// parseConfig parses command line flags and reads the configuration file.
func parseConfig() (configapp.Config, error) {
	var fileName string
	flag.StringVar(&fileName, "f", "", "Configuration file")
	flag.Parse()

	if fileName == "" {
		flag.Usage()
		return configapp.Config{}, ErrConfigFileNotProvided
	}

	cfg, err := configapp.ReadYamlCnxFile(fileName)
	if err != nil {
		return configapp.Config{}, fmt.Errorf("error reading configuration file: %w", err)
	}
	return cfg, nil
}

// initInfrastructure initializes S3 client and database connection.
func initInfrastructure(ctx context.Context, cfg configapp.Config, l *slog.Logger) (*s3.Client, *sql.DB, error) {
	s3Client, err := initS3Client(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("error initializing S3 client: %w", err)
	}

	dbConn, err := dbinit.InitializeDatabase(ctx, cfg.Database.URL, l)
	if err != nil {
		return nil, nil, fmt.Errorf("error initializing database: %w", err)
	}

	return s3Client, dbConn, nil
}

// initServices creates and configures all services.
func initServices(
	cfg configapp.Config, s3Client *s3.Client, dbConn *sql.DB, l *slog.Logger,
) (*dbsvc.Service, *scanner.Service, *scheduler.Scheduler) {
	dbService := dbsvc.NewService(cfg, dbConn)
	dbService.SetLogger(l)

	scannerService := scanner.NewService(cfg, s3Client, dbConn)
	scannerService.SetLogger(l)

	scheduler := scheduler.NewScheduler(cfg, dbConn, scannerService)
	scheduler.SetLogger(l)

	return dbService, scannerService, scheduler
}

// performInitialScan runs the initial bucket scan if enabled.
func performInitialScan(ctx context.Context, cfg configapp.Config, scannerService *scanner.Service, l *slog.Logger) {
	if !cfg.Scan.EnableInitialScan {
		return
	}

	if scannerService == nil {
		l.Warn("Skipping initial scan - scanner service not available (database unavailable)")
		return
	}

	l.Info("Performing initial bucket scan")
	if err := scannerService.DiscoverAndScanAllBuckets(ctx); err != nil {
		l.Error("error during initial scan", slog.String("error", err.Error()))
		// Don't exit - continue with application startup even if initial scan fails
	} else {
		l.Info("Initial bucket scan completed successfully")
	}
}

// shutdown handles graceful shutdown of services.
func shutdown(s *app.App, scheduler *scheduler.Scheduler, l *slog.Logger) {
	l.Info("stop the server")
	if scheduler != nil {
		scheduler.Stop()
	}
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
	if configApp.S3.Endpoint != "" {
		// Check if this is an AWS S3 endpoint (contains amazonaws.com)
		isAwsEndpoint := strings.Contains(configApp.S3.Endpoint, "amazonaws.com")
		usePathStyle := !isAwsEndpoint

		// fmt.Printf("Custom endpoint detected - AWS: %t, UsePathStyle: %t\n", isAwsEndpoint, usePathStyle)

		// Use functional options pattern to configure the S3 client
		return s3.NewFromConfig(cfg, func(o *s3.Options) {
			// Set the custom endpoint URL
			o.BaseEndpoint = aws.String(configApp.S3.Endpoint)
			// Use path-style addressing only for non-AWS endpoints (like MinIO)
			// AWS S3 should use virtual-hosted-style (UsePathStyle = false)
			o.UsePathStyle = usePathStyle
			// Ensure region is set correctly for both AWS and custom endpoints
			o.Region = configApp.S3.Region
		}), nil
	}

	// Standard AWS S3 client configuration
	// For AWS S3, we need to ensure the region is properly set
	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.Region = configApp.S3.Region
	}), nil
}

// GetAwsConfig returns an aws.Config based on the provided configuration.
func GetAwsConfig(ctx context.Context, cfgApp configapp.Config) (aws.Config, error) {
	// Initialize an empty config
	var cfg aws.Config

	if cfgApp.S3.Endpoint != "" {
		// Parse the endpoint URL for validation
		_, err := url.Parse(cfgApp.S3.Endpoint)
		if err != nil {
			return aws.Config{}, fmt.Errorf("invalid S3 endpoint URL: %w", err)
		}

		// Load basic configuration with region & credentials
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(cfgApp.S3.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfgApp.S3.AccessKey,
				cfgApp.S3.APIKey,
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

	if cfgApp.S3.SsoAwsProfile != "" {
		cfg, err := config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(cfgApp.S3.SsoAwsProfile))
		if err != nil {
			// s.log.Error("Error loading SSO profile", slog.String("error", err.Error()))
			return cfg, fmt.Errorf("error loading SSO profile: %w", err)
		}
		// s.log.Debug("SSO profile loaded")
		return cfg, nil
	}

	if cfgApp.S3.AccessKey != "" && cfgApp.S3.APIKey != "" {
		cfg, err := config.LoadDefaultConfig(ctx,
			config.WithRegion(cfgApp.S3.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfgApp.S3.AccessKey,
				cfgApp.S3.APIKey,
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
		config.WithRegion(cfgApp.S3.Region),
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
