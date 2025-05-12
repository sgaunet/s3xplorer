package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gorilla/mux"
	"github.com/sgaunet/s3xplorer/pkg/config"
	"github.com/sgaunet/s3xplorer/pkg/s3svc"
)

// App is the main structure of the application
type App struct {
	cfg         config.Config
	awsS3Client *s3.Client
	s3svc       *s3svc.Service
	router      *mux.Router
	srv         *http.Server
	log         *slog.Logger
}

// emptyLogger returns a logger that discards all log entries
func emptyLogger() *slog.Logger {
	// Use DiscardHandler to create a logger that doesn't output anything
	return slog.New(slog.DiscardHandler)
}

// NewApp creates a new App
// NewApp initializes the S3 client and launch the web server in a goroutine
// By default the logger is set to write to /dev/null
func NewApp(cfg config.Config, s3Client *s3.Client) *App {
	s := &App{
		cfg:         cfg,
		awsS3Client: s3Client,
		router:      mux.NewRouter().StrictSlash(true),
		log:         emptyLogger(),
		srv:         &http.Server{},
		s3svc:       s3svc.NewS3Svc(cfg, s3Client),
	}

	s.initRouter()
	// Start the web server in a goroutine
	go func() {
		err := s.startWebServer()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			// Log the error but don't return it as the server is running in a goroutine
			s.log.Error("server error", slog.String("error", err.Error()))
		}
	}()

	return s
}

// SetLogger sets the logger of the App
func (s *App) SetLogger(l *slog.Logger) {
	s.log = l
	s.s3svc.SetLogger(l)
}

// StopServer stops the web server
func (s *App) StopServer() error {
	if err := s.srv.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("error stopping server: %w", err)
	}
	return nil
}

// Router returns the router of the App
func (s App) Router() http.Handler {
	return s.router
}

// startWebServer starts the web server
// Default port is 8081
func (s *App) startWebServer() error {
	// Define constants for server configuration
	const (
		// DefaultServerPort is the default port for the web server
		DefaultServerPort = "8081"
		// DefaultReadHeaderTimeout is the timeout for reading request headers
		DefaultReadHeaderTimeout = 5 * time.Second
	)

	// Set a read header timeout to mitigate Slowloris attacks
	s.srv = &http.Server{
		Addr:              ":" + DefaultServerPort,
		Handler:           s.router,
		ReadHeaderTimeout: DefaultReadHeaderTimeout,
	}
	s.log.Info("Starting server", slog.String("addr", s.srv.Addr))
	err := s.srv.ListenAndServe()
	if err != nil {
		return fmt.Errorf("error starting server: %w", err)
	}
	return nil
}
