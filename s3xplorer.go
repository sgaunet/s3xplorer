package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/sgaunet/s3xplorer/pkg/app"
	"github.com/sgaunet/s3xplorer/pkg/config"
)

func main() {
	var err error
	var fileName string
	var cfg config.Config
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
	if cfg, err = config.ReadYamlCnxFile(fileName); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading configuration file: %s\n", err.Error())
		os.Exit(1)
	}
	// Initialize the logger
	l := initTrace(cfg.LogLevel)

	// Handle SIGTERM/SIGINT
	ctx, cancelFunc := context.WithCancel(context.Background())
	SetupCloseHandler(ctx, cancelFunc, l)

	// Create the app
	s, err := app.NewApp(cfg)
	if err != nil {
		l.Error("error creating the app: %s", slog.String("error", err.Error()))
		os.Exit(1)
	}
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
