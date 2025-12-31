package s3svc

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// UploadObject uploads a single object to S3.
// Parameters:
//   - ctx: Context for the request
//   - key: S3 object key (full path including filename)
//   - body: io.Reader containing the file data
//   - contentType: MIME type of the file (e.g., "image/jpeg", "application/pdf")
//   - size: Size of the file in bytes (for progress tracking and validation)
func (s *Service) UploadObject(
	ctx context.Context,
	key string,
	body io.Reader,
	contentType string,
	size int64,
) error {
	input := &s3.PutObjectInput{
		Bucket:        &s.cfg.S3.Bucket,
		Key:           &key,
		Body:          body,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	}

	_, err := s.awsS3Client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("UploadObject: error uploading to S3: %w", err)
	}

	s.log.Debug("UploadObject completed",
		slog.String("key", key),
		slog.String("contentType", contentType),
		slog.Int64("size", size))

	return nil
}
