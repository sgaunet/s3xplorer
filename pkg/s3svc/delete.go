package s3svc

import (
	"context"
	"crypto/md5" //nolint:gosec // MD5 required by S3 API for Content-MD5 header, not for cryptographic security
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// DeleteObject deletes a single object from S3.
// Parameters:
//   - ctx: Context for the request
//   - key: S3 object key to delete
func (s *Service) DeleteObject(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: &s.cfg.S3.Bucket,
		Key:    &key,
	}

	_, err := s.awsS3Client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("DeleteObject: error deleting from S3: %w", err)
	}

	s.log.Debug("DeleteObject completed", slog.String("key", key))
	return nil
}

// deletePayload represents the XML structure for DeleteObjects request body.
// This is used to compute the Content-MD5 header required by some S3-compatible services.
type deletePayload struct {
	XMLName xml.Name       `xml:"Delete"`
	Objects []deleteObject `xml:"Object"`
	Quiet   bool           `xml:"Quiet"`
}

type deleteObject struct {
	Key string `xml:"Key"`
}

// computeDeleteContentMD5 computes the MD5 hash of the DeleteObjects request body.
// This is required by MinIO and some S3-compatible services.
func computeDeleteContentMD5(objects []types.ObjectIdentifier, quiet bool) (string, error) {
	// Build the XML payload
	payload := deletePayload{
		Objects: make([]deleteObject, len(objects)),
		Quiet:   quiet,
	}
	for i, obj := range objects {
		payload.Objects[i] = deleteObject{Key: *obj.Key}
	}

	// Marshal to XML
	xmlBytes, err := xml.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal delete payload: %w", err)
	}

	// Compute MD5 hash
	hash := md5.Sum(xmlBytes) //nolint:gosec // MD5 required by S3 API for Content-MD5 header
	// Encode to base64 as required by S3
	return base64.StdEncoding.EncodeToString(hash[:]), nil
}

// addContentMD5Middleware creates a middleware that adds the Content-MD5 header to the request.
func addContentMD5Middleware(contentMD5 string) func(*s3.Options) {
	return func(o *s3.Options) {
		o.APIOptions = append(o.APIOptions, func(stack *middleware.Stack) error {
			return stack.Finalize.Add(
				middleware.FinalizeMiddlewareFunc(
					"AddContentMD5",
					func(
						ctx context.Context,
						in middleware.FinalizeInput,
						next middleware.FinalizeHandler,
					) (middleware.FinalizeOutput, middleware.Metadata, error) {
						req, ok := in.Request.(*smithyhttp.Request)
						if ok {
							req.Header.Set("Content-MD5", contentMD5)
						}
						return next.HandleFinalize(ctx, in)
					},
				),
				middleware.Before,
			)
		})
	}
}

// DeleteObjects deletes multiple objects from S3 in a single batch operation.
// S3 supports up to 1000 objects per batch request.
// Parameters:
//   - ctx: Context for the request
//   - keys: Slice of S3 object keys to delete
func (s *Service) DeleteObjects(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil // Nothing to delete
	}

	// AWS S3 DeleteObjects API has a limit of 1000 objects per request
	const maxBatchSize = 1000
	if len(keys) > maxBatchSize {
		//nolint:err113 // Dynamic error provides useful context about batch size violation
		return fmt.Errorf("DeleteObjects: too many keys (%d), maximum is %d", len(keys), maxBatchSize)
	}

	// Convert string keys to ObjectIdentifier structs
	objects := make([]types.ObjectIdentifier, len(keys))
	for i, key := range keys {
		keyCopy := key // Create copy for pointer
		objects[i] = types.ObjectIdentifier{
			Key: &keyCopy,
		}
	}

	quiet := false
	input := &s3.DeleteObjectsInput{
		Bucket: &s.cfg.S3.Bucket,
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(quiet), // Get detailed response
		},
	}

	// Compute Content-MD5 header for MinIO compatibility
	contentMD5, err := computeDeleteContentMD5(objects, quiet)
	if err != nil {
		return fmt.Errorf("DeleteObjects: failed to compute Content-MD5: %w", err)
	}

	// Add Content-MD5 header using middleware
	output, err := s.awsS3Client.DeleteObjects(ctx, input, addContentMD5Middleware(contentMD5))
	if err != nil {
		return fmt.Errorf("DeleteObjects: error deleting from S3: %w", err)
	}

	// Check for partial failures
	if len(output.Errors) > 0 {
		s.log.Warn("DeleteObjects: some objects failed to delete",
			slog.Int("failed", len(output.Errors)),
			slog.Int("total", len(keys)))
		for _, deleteError := range output.Errors {
			s.log.Error("Failed to delete object",
				slog.String("key", *deleteError.Key),
				slog.String("code", *deleteError.Code),
				slog.String("message", *deleteError.Message))
		}
		//nolint:err113 // Dynamic error provides useful context about partial deletion failures
		return fmt.Errorf("DeleteObjects: %d of %d objects failed to delete", len(output.Errors), len(keys))
	}

	s.log.Debug("DeleteObjects completed",
		slog.Int("count", len(keys)),
		slog.Int("deleted", len(output.Deleted)))

	return nil
}
