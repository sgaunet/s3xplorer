package s3svc

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sgaunet/s3xplorer/pkg/dto"
)

// ListBuckets returns a list of all S3 buckets accessible with the current credentials.
func (s *Service) ListBuckets(ctx context.Context) ([]dto.Bucket, error) {
	s.log.Debug("Listing buckets")
	
	// Call S3 ListBuckets API
	output, err := s.awsS3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		s.log.Error("Failed to list buckets", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}
	
	// Convert to our DTO type
	buckets := make([]dto.Bucket, 0, len(output.Buckets))
	for _, bucket := range output.Buckets {
		buckets = append(buckets, dto.Bucket{
			Name:         *bucket.Name,
			CreationDate: *bucket.CreationDate,
		})
	}
	
	s.log.Debug("Listed buckets", slog.Int("count", len(buckets)))
	return buckets, nil
}

// IsBucketEmpty checks if a bucket is empty (has no objects).
func (s *Service) IsBucketEmpty(ctx context.Context) (bool, error) {
	var maxKeys int32 = 1
	input := &s3.ListObjectsV2Input{
		Bucket:  &s.cfg.Bucket,
		MaxKeys: &maxKeys,
	}

	if s.cfg.Prefix != "" {
		input.Prefix = &s.cfg.Prefix
	}

	result, err := s.awsS3Client.ListObjectsV2(ctx, input)
	if err != nil {
		s.log.Error("Failed to check if bucket is empty",
			slog.String("bucket", s.cfg.Bucket),
			slog.String("error", err.Error()))
		return false, fmt.Errorf("failed to check if bucket is empty: %w", err)
	}

	return len(result.Contents) == 0, nil
}

// SwitchBucket updates the current bucket in the service configuration.
func (s *Service) SwitchBucket(bucketName string) {
	s.log.Info("Switching bucket", 
		slog.String("from", s.cfg.Bucket), 
		slog.String("to", bucketName))
	s.cfg.Bucket = bucketName
	// Reset prefix when switching buckets
	s.cfg.Prefix = ""
}

// GetBucketName returns the current bucket name.
func (s *Service) GetBucketName() string {
	return s.cfg.Bucket
}
