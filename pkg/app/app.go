package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
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

// emptyLogger returns a logger that writes to /dev/null
func emptyLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// NewApp creates a new App
// NewApp initializes the S3 client and launch the web server in a goroutine
// By default the logger is set to write to /dev/null
func NewApp(cfg config.Config) (*App, error) {
	s := &App{
		cfg:    cfg,
		router: mux.NewRouter().StrictSlash(true),
		log:    emptyLogger(),
		srv:    &http.Server{},
	}
	err := s.initS3Client()
	if err != nil {
		return s, err
	}
	s.s3svc = s3svc.NewS3Svc(cfg, s.awsS3Client)

	s.initRouter()
	go s.startWebServer()

	return s, err
}

// SetLogger sets the logger of the App
func (s *App) SetLogger(l *slog.Logger) {
	s.log = l
	s.s3svc.SetLogger(l)
}

// startWebServer starts the web server
// Default port is 8081
func (s *App) startWebServer() error {
	s.srv.Addr = ":8081"
	s.log.Info("Starting server", slog.String("addr", s.srv.Addr))
	err := s.srv.ListenAndServe()
	if err != nil {
		return fmt.Errorf("error starting server: %w", err)
	}
	return nil
}

// StopServer stops the web server
func (s *App) StopServer() error {
	if err := s.srv.Shutdown(context.Background()); err != nil {
		return fmt.Errorf("error stopping server: %w", err)
	}
	return nil
}

// initS3Client initializes the S3 client
func (s *App) initS3Client() (err error) {
	var cfg aws.Config
	cfg, err = s.GetAwsConfig()
	if err != nil {
		return fmt.Errorf("error getting AWS config: %w", err)
	}
	// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3
	s.awsS3Client = s3.NewFromConfig(cfg)
	return nil
}

// Router returns the router of the App
func (s App) Router() http.Handler {
	return s.router
}
