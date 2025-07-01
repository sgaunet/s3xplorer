package dbsvc

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/sgaunet/s3xplorer/pkg/config"
	"github.com/sgaunet/s3xplorer/pkg/database"
	"github.com/sgaunet/s3xplorer/pkg/dto"
)

// Service provides database operations for S3 objects
type Service struct {
	db      *sql.DB
	queries *database.Queries
	cfg     config.Config
	log     *slog.Logger
}

// NewService creates a new database service
func NewService(cfg config.Config, db *sql.DB) *Service {
	return &Service{
		db:      db,
		queries: database.New(db),
		cfg:     cfg,
		log:     slog.New(slog.DiscardHandler),
	}
}

// SetLogger sets the logger for the service
func (s *Service) SetLogger(log *slog.Logger) {
	s.log = log
}

// GetBuckets returns all available buckets
func (s *Service) GetBuckets(ctx context.Context) ([]dto.Bucket, error) {
	buckets, err := s.queries.ListBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	result := make([]dto.Bucket, len(buckets))
	for i, bucket := range buckets {
		result[i] = dto.Bucket{
			Name:   bucket.Name,
			Region: bucket.Region.String,
		}
	}

	return result, nil
}

// GetFolders returns folders at the specified prefix
func (s *Service) GetFolders(ctx context.Context, bucketName, prefix string, limit, offset int) ([]dto.S3Object, error) {
	bucket, err := s.queries.GetBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %w", err)
	}

	objects, err := s.queries.ListS3Folders(ctx, database.ListS3FoldersParams{
		BucketID: bucket.ID,
		Column2:  prefix,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	return s.convertToDTO(objects), nil
}

// GetObjects returns objects at the specified prefix
func (s *Service) GetObjects(ctx context.Context, bucketName, prefix string, limit, offset int) ([]dto.S3Object, error) {
	bucket, err := s.queries.GetBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %w", err)
	}

	objects, err := s.queries.ListS3Objects(ctx, database.ListS3ObjectsParams{
		BucketID: bucket.ID,
		Column2:  prefix,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	return s.convertToDTO(objects), nil
}

// SearchObjects searches for objects matching the query
func (s *Service) SearchObjects(ctx context.Context, bucketName, query string, limit, offset int) ([]dto.S3Object, error) {
	bucket, err := s.queries.GetBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %w", err)
	}

	objects, err := s.queries.SearchS3Objects(ctx, database.SearchS3ObjectsParams{
		BucketID: bucket.ID,
		Column2:  sql.NullString{String: query, Valid: true},
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search objects: %w", err)
	}

	return s.convertToDTO(objects), nil
}

// GetObjectsByPrefix returns objects with the specified prefix pattern
func (s *Service) GetObjectsByPrefix(ctx context.Context, bucketName, prefix string, limit, offset int) ([]dto.S3Object, error) {
	bucket, err := s.queries.GetBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %w", err)
	}

	objects, err := s.queries.ListS3ObjectsByPrefix(ctx, database.ListS3ObjectsByPrefixParams{
		BucketID: bucket.ID,
		Column2:  sql.NullString{String: prefix, Valid: true},
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects by prefix: %w", err)
	}

	return s.convertToDTO(objects), nil
}

// CountObjects returns the total count of objects matching the criteria
func (s *Service) CountObjects(ctx context.Context, bucketName, prefix string) (int64, error) {
	bucket, err := s.queries.GetBucket(ctx, bucketName)
	if err != nil {
		return 0, fmt.Errorf("bucket not found: %w", err)
	}

	count, err := s.queries.CountS3Objects(ctx, database.CountS3ObjectsParams{
		BucketID: bucket.ID,
		Column2:  prefix,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to count objects: %w", err)
	}

	return count, nil
}

// convertToDTO converts database objects to DTO objects
func (s *Service) convertToDTO(objects []database.S3Object) []dto.S3Object {
	result := make([]dto.S3Object, len(objects))
	for i, obj := range objects {
		result[i] = dto.S3Object{
			Key:          obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified.Time,
			ETag:         obj.Etag.String,
			StorageClass: obj.StorageClass.String,
			IsFolder:     obj.IsFolder.Bool,
			Prefix:       obj.Prefix.String,
		}
		
		// Format size for display
		result[i].SizeHuman = s.formatSize(obj.Size)
		
		// Extract filename from key
		if idx := strings.LastIndex(obj.Key, "/"); idx != -1 {
			result[i].Name = obj.Key[idx+1:]
		} else {
			result[i].Name = obj.Key
		}
	}
	return result
}

// formatSize formats file size in human readable format
func (s *Service) formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return strconv.FormatInt(size, 10) + " B"
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}