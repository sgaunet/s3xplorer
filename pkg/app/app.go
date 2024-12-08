package app

import (
	"context"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gorilla/mux"
	"github.com/sgaunet/s3xplorer/pkg/config"
	"github.com/sgaunet/s3xplorer/pkg/s3svc"
	"github.com/sgaunet/s3xplorer/pkg/views"
	"github.com/sirupsen/logrus"
)

type App struct {
	cfg         config.Config
	awsS3Client *s3.Client
	s3svc       *s3svc.Service
	router      *mux.Router
	views       *views.Views
	srv         *http.Server
	log         *logrus.Logger
}

func NewApp(cfg config.Config) (*App, error) {
	s := &App{
		cfg:    cfg,
		router: mux.NewRouter().StrictSlash(true),
		views:  views.NewViews(),
		log:    initTrace(cfg.LogLevel),
		srv:    &http.Server{},
	}
	err := s.initS3Client()
	if err != nil {
		return s, err
	}
	s3svc := s3svc.NewS3Svc(cfg, s.awsS3Client)
	s.s3svc = s3svc

	s.initRouter()
	go s.startWebServer()

	return s, err
}

func (s *App) startWebServer() {
	s.srv.Addr = ":8081"
	s.log.Infoln("listen :8081")
	s.srv.ListenAndServe()
}

func (s *App) StopServer() {
	if err := s.srv.Shutdown(context.Background()); err != nil {
		s.log.Fatal(err)
	}
}

func (s *App) initS3Client() (err error) {
	var cfg aws.Config
	cfg, err = s.GetAwsConfig()
	if err != nil {
		return
	}
	// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3
	s.awsS3Client = s3.NewFromConfig(cfg)
	return nil
}

func (s App) Router() http.Handler {
	return s.router
}

func initTrace(debugLevel string) *logrus.Logger {
	appLog := logrus.New()
	appLog.SetFormatter(&logrus.TextFormatter{
		DisableColors:    false,
		FullTimestamp:    false,
		DisableTimestamp: true,
	})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	appLog.SetOutput(os.Stdout)

	switch debugLevel {
	case "info":
		appLog.SetLevel(logrus.InfoLevel)
	case "warn":
		appLog.SetLevel(logrus.WarnLevel)
	case "error":
		appLog.SetLevel(logrus.ErrorLevel)
	default:
		appLog.SetLevel(logrus.DebugLevel)
	}
	return appLog
}
