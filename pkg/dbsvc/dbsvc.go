package dbsvc

import (
	"context"
	"database/sql"
	"errors"
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
		
		// Set download availability based on storage class
		// Objects are downloadable if they are in STANDARD storage or if storage class is empty
		// For Glacier objects, they would need to be restored first (not implemented in DB service)
		storageClass := obj.StorageClass.String
		result[i].IsDownloadable = (storageClass == "" || storageClass == "STANDARD")
		result[i].IsRestoring = false // DB doesn't track restore status, assume false
		
		// Extract filename/folder name from key
		if obj.IsFolder.Bool {
			// For folders, remove trailing slash and get the last part
			folderKey := strings.TrimSuffix(obj.Key, "/")
			if idx := strings.LastIndex(folderKey, "/"); idx != -1 {
				result[i].Name = folderKey[idx+1:]
			} else {
				result[i].Name = folderKey
			}
		} else {
			// For files, get the last part after the slash
			if idx := strings.LastIndex(obj.Key, "/"); idx != -1 {
				result[i].Name = obj.Key[idx+1:]
			} else {
				result[i].Name = obj.Key
			}
		}
	}
	return result
}

// GetDirectChildren returns only immediate children (non-recursive) for hierarchical navigation
func (s *Service) GetDirectChildren(ctx context.Context, bucketName, prefix string, limit, offset int) ([]dto.S3Object, error) {
	bucket, err := s.queries.GetBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %w", err)
	}

	objects, err := s.queries.GetDirectChildren(ctx, database.GetDirectChildrenParams{
		BucketID: bucket.ID,
		Column2:  prefix,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get direct children: %w", err)
	}

	return s.convertToDTO(objects), nil
}

// GetBreadcrumbPath returns parent folders for breadcrumb navigation
func (s *Service) GetBreadcrumbPath(ctx context.Context, bucketName, currentPath string) ([]dto.S3Object, error) {
	bucket, err := s.queries.GetBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %w", err)
	}

	if currentPath == "" {
		return []dto.S3Object{}, nil
	}

	breadcrumbs, err := s.queries.GetBreadcrumbPath(ctx, database.GetBreadcrumbPathParams{
		BucketID: bucket.ID,
		Key:      currentPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get breadcrumb path: %w", err)
	}

	return s.convertToDTO(breadcrumbs), nil
}

// GetParentFolder returns the parent folder of the given path
func (s *Service) GetParentFolder(ctx context.Context, bucketName, folderPath string) (*dto.S3Object, error) {
	if folderPath == "" {
		return nil, nil
	}

	bucket, err := s.queries.GetBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %w", err)
	}

	parentFolder, err := s.queries.GetParentFolder(ctx, database.GetParentFolderParams{
		BucketID: bucket.ID,
		Key:      folderPath,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get parent folder: %w", err)
	}

	converted := s.convertToDTO([]database.S3Object{parentFolder})
	if len(converted) > 0 {
		return &converted[0], nil
	}
	return nil, nil
}

// BuildBreadcrumbs creates breadcrumb navigation from a path
func (s *Service) BuildBreadcrumbs(path string) []dto.Breadcrumb {
	if path == "" {
		return []dto.Breadcrumb{{Name: "Root", Path: ""}}
	}

	var breadcrumbs []dto.Breadcrumb
	breadcrumbs = append(breadcrumbs, dto.Breadcrumb{Name: "Root", Path: ""})

	parts := strings.Split(strings.Trim(path, "/"), "/")
	currentPath := ""

	for _, part := range parts {
		if part == "" {
			continue
		}
		if currentPath != "" {
			currentPath += "/"
		}
		currentPath += part
		breadcrumbs = append(breadcrumbs, dto.Breadcrumb{
			Name: part,
			Path: currentPath + "/",
		})
	}

	return breadcrumbs
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