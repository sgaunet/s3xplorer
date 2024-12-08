package s3svc

import (
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sgaunet/s3xplorer/pkg/config"
	"github.com/sirupsen/logrus"
)

// Service is the struct for the S3 service
type Service struct {
	cfg         config.Config
	awsS3Client *s3.Client
	log         *logrus.Logger
}

// NewS3Svc creates a new S3 service
func NewS3Svc(cfg config.Config, S3Client *s3.Client) *Service {
	s := &Service{
		cfg:         cfg,
		awsS3Client: S3Client,
		log:         logrus.New(),
	}
	return s
}

// SetLogger sets the logger
func (s *Service) SetLogger(log *logrus.Logger) {
	s.log = log
}
