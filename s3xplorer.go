package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
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
	s.StopServer()
}

func SetupCloseHandler(ctx context.Context, cancelFunc context.CancelFunc, log *slog.Logger) {
	c := make(chan os.Signal, 5)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		s := <-c
		log.Info("INFO: signal received", slog.String("signal", s.String()))
		cancelFunc()
	}()
}

// initTrace initializes the logger
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

// initS3Client initializes the S3 client
func initS3Client(ctx context.Context, configApp configapp.Config) (*s3.Client, error) {
	var cfg aws.Config
	cfg, err := GetAwsConfig(ctx, configApp)
	if err != nil {
		return nil, fmt.Errorf("error getting AWS config: %w", err)
	}
	// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3
	return s3.NewFromConfig(cfg), nil
}

// GetAwsConfig returns an aws.Config
func GetAwsConfig(ctx context.Context, cfgApp configapp.Config) (cfg aws.Config, err error) {
	if cfgApp.S3endpoint != "" {
		staticResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
			return aws.Endpoint{
				PartitionID:       "aws",
				URL:               cfgApp.S3endpoint, // or where ever you ran minio
				SigningRegion:     cfgApp.S3Region,
				HostnameImmutable: true,
			}, nil
		})

		cfg = aws.Config{
			Region:           cfgApp.S3Region,
			Credentials:      credentials.NewStaticCredentialsProvider(cfgApp.S3ApikKey, cfgApp.S3accessKey, ""),
			EndpointResolver: staticResolver,
		}
		return
	}

	if cfgApp.SsoAwsProfile != "" {
		cfg, err = config.LoadDefaultConfig(ctx, config.WithSharedConfigProfile(cfgApp.SsoAwsProfile))
		if err != nil {
			// s.log.Error("Error loading SSO profile", slog.String("error", err.Error()))
			return cfg, fmt.Errorf("error loading SSO profile: %w", err)
		}
		// s.log.Debug("SSO profile loaded")
		return cfg, nil
	}

	if cfgApp.S3ApikKey == "" && cfgApp.S3accessKey == "" {
		cfg, err = config.LoadDefaultConfig(ctx, config.WithRegion(cfgApp.S3Region))
		if err != nil {
			// s.log.Error("Error loading default config", slog.String("error", err.Error()))
			return cfg, fmt.Errorf("error loading default config: %w", err)
		}
		// s.log.Debug("Default config loaded")
		return cfg, nil
	}
	return cfg, errors.New("no method to initialize aws.Config")
}
