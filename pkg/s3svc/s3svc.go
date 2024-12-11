package s3svc

import (
	"io"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sgaunet/s3xplorer/pkg/config"
)

// Service is the struct for the S3 service
type Service struct {
	cfg         config.Config
	awsS3Client *s3.Client
	log         *slog.Logger
}

// NewS3Svc creates a new S3 service
// It requires a config.Config and a *s3.Client
// By default the logger is set to write to /dev/null
func NewS3Svc(cfg config.Config, S3Client *s3.Client) *Service {
	s := &Service{
		cfg:         cfg,
		awsS3Client: S3Client,
		log:         slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	return s
}

// SetLogger sets the logger
func (s *Service) SetLogger(log *slog.Logger) {
	s.log = log
}
