// Package dbsvc provides database service operations for S3 object metadata.
package dbsvc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/sgaunet/s3xplorer/pkg/config"
	"github.com/sgaunet/s3xplorer/pkg/database"
	"github.com/sgaunet/s3xplorer/pkg/dto"
)

// ErrNoParentFolder is returned when there is no parent folder.
var ErrNoParentFolder = errors.New("no parent folder")

// Service provides database operations for S3 objects.
type Service struct {
	db      *sql.DB
	queries *database.Queries
	cfg     config.Config
	log     *slog.Logger
}

// NewService creates a new database service.
func NewService(cfg config.Config, db *sql.DB) *Service {
	return &Service{
		db:      db,
		queries: database.New(db),
		cfg:     cfg,
		log:     slog.New(slog.DiscardHandler),
	}
}

// SetLogger sets the logger for the service.
func (s *Service) SetLogger(log *slog.Logger) {
	s.log = log
}

// GetDB returns the underlying database connection.
func (s *Service) GetDB() *sql.DB {
	return s.db
}

// GetBuckets returns only accessible buckets for normal user operations.
func (s *Service) GetBuckets(ctx context.Context) ([]dto.Bucket, error) {
	buckets, err := s.queries.ListAccessibleBuckets(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list accessible buckets: %w", err)
	}

	result := make([]dto.Bucket, len(buckets))
	for i, bucket := range buckets {
		result[i] = s.convertBucketToDTO(bucket, "", "", nil)
	}

	return result, nil
}

// GetBucketsWithStatus returns all buckets with detailed status information for admin/debug purposes.
func (s *Service) GetBucketsWithStatus(ctx context.Context) ([]dto.Bucket, error) {
	buckets, err := s.queries.ListBucketsWithStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets with status: %w", err)
	}

	result := make([]dto.Bucket, len(buckets))
	for i, bucketRow := range buckets {
		scanStatus := "never_scanned"
		if bucketRow.LatestScanStatus != "" {
			scanStatus = bucketRow.LatestScanStatus
		}
		
		scanError := ""
		if bucketRow.LatestScanError.Valid {
			scanError = bucketRow.LatestScanError.String
		}
		
		var scanCompletedAt *time.Time
		if bucketRow.LatestScanCompletedAt.Valid {
			scanCompletedAt = &bucketRow.LatestScanCompletedAt.Time
		}
		
		// Convert the row to a Bucket struct for the helper function
		bucket := database.Bucket{
			ID:               bucketRow.ID,
			Name:             bucketRow.Name,
			Region:           bucketRow.Region,
			CreatedAt:        bucketRow.CreatedAt,
			UpdatedAt:        bucketRow.UpdatedAt,
			MarkedForDeletion: bucketRow.MarkedForDeletion,
			LastAccessibleAt: bucketRow.LastAccessibleAt,
			AccessError:      bucketRow.AccessError,
		}
		
		result[i] = s.convertBucketToDTO(bucket, scanStatus, scanError, scanCompletedAt)
	}

	return result, nil
}

// GetFolders returns folders at the specified prefix.
func (s *Service) GetFolders(
	ctx context.Context, bucketName, prefix string, limit, offset int,
) ([]dto.S3Object, error) {
	bucket, err := s.queries.GetBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %w", err)
	}

	objects, err := s.queries.ListS3Folders(ctx, database.ListS3FoldersParams{
		BucketID: bucket.ID,
		Column2:  prefix,
		Limit:    int32(min(int64(limit), math.MaxInt32)),   //nolint:gosec
		Offset:   int32(min(int64(offset), math.MaxInt32)), //nolint:gosec
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	return s.convertToDTO(objects), nil
}

// GetObjects returns objects at the specified prefix.
func (s *Service) GetObjects(
	ctx context.Context, bucketName, prefix string, limit, offset int,
) ([]dto.S3Object, error) {
	bucket, err := s.queries.GetBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %w", err)
	}

	objects, err := s.queries.ListS3Objects(ctx, database.ListS3ObjectsParams{
		BucketID: bucket.ID,
		Column2:  prefix,
		Limit:    int32(min(int64(limit), math.MaxInt32)),   //nolint:gosec
		Offset:   int32(min(int64(offset), math.MaxInt32)), //nolint:gosec
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects: %w", err)
	}

	return s.convertToDTO(objects), nil
}

// SearchObjects searches for objects matching the query.
func (s *Service) SearchObjects(
	ctx context.Context, bucketName, query string, limit, offset int,
) ([]dto.S3Object, error) {
	bucket, err := s.queries.GetBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %w", err)
	}

	objects, err := s.queries.SearchS3Objects(ctx, database.SearchS3ObjectsParams{
		BucketID: bucket.ID,
		Column2:  sql.NullString{String: query, Valid: true},
		Limit:    int32(min(int64(limit), math.MaxInt32)),   //nolint:gosec
		Offset:   int32(min(int64(offset), math.MaxInt32)), //nolint:gosec
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search objects: %w", err)
	}

	return s.convertToDTO(objects), nil
}

// GetObjectsByPrefix returns objects with the specified prefix pattern.
func (s *Service) GetObjectsByPrefix(
	ctx context.Context, bucketName, prefix string, limit, offset int,
) ([]dto.S3Object, error) {
	bucket, err := s.queries.GetBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %w", err)
	}

	objects, err := s.queries.ListS3ObjectsByPrefix(ctx, database.ListS3ObjectsByPrefixParams{
		BucketID: bucket.ID,
		Column2:  sql.NullString{String: prefix, Valid: true},
		Limit:    int32(min(int64(limit), math.MaxInt32)),   //nolint:gosec
		Offset:   int32(min(int64(offset), math.MaxInt32)), //nolint:gosec
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list objects by prefix: %w", err)
	}

	return s.convertToDTO(objects), nil
}

// CountObjects returns the total count of objects matching the criteria.
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

// GetDirectChildren returns only immediate children (non-recursive) for hierarchical navigation.
func (s *Service) GetDirectChildren(
	ctx context.Context, bucketName, prefix string, limit, offset int,
) ([]dto.S3Object, error) {
	bucket, err := s.queries.GetBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %w", err)
	}

	objects, err := s.queries.GetDirectChildren(ctx, database.GetDirectChildrenParams{
		BucketID: bucket.ID,
		Column2:  prefix,
		Limit:    int32(min(int64(limit), math.MaxInt32)),   //nolint:gosec
		Offset:   int32(min(int64(offset), math.MaxInt32)), //nolint:gosec
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get direct children: %w", err)
	}

	return s.convertToDTO(objects), nil
}

// GetBreadcrumbPath returns parent folders for breadcrumb navigation.
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

// GetParentFolder returns the parent folder of the given path.
func (s *Service) GetParentFolder(ctx context.Context, bucketName, folderPath string) (*dto.S3Object, error) {
	if folderPath == "" {
		return nil, ErrNoParentFolder
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
			return nil, ErrNoParentFolder
		}
		return nil, fmt.Errorf("failed to get parent folder: %w", err)
	}

	converted := s.convertToDTO([]database.S3Object{parentFolder})
	if len(converted) > 0 {
		return &converted[0], nil
	}
	return nil, ErrNoParentFolder
}

// BuildBreadcrumbs creates breadcrumb navigation from a path.
func (s *Service) BuildBreadcrumbs(path string) []dto.Breadcrumb {
	if path == "" {
		return []dto.Breadcrumb{{Name: "Root", Path: ""}}
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	breadcrumbs := make([]dto.Breadcrumb, 0, len(parts)+1)
	breadcrumbs = append(breadcrumbs, dto.Breadcrumb{Name: "Root", Path: ""})
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

// convertBucketToDTO converts a database bucket to DTO with accessibility status.
func (s *Service) convertBucketToDTO(
	bucket database.Bucket, scanStatus, scanError string, scanCompletedAt *time.Time,
) dto.Bucket {
	isAccessible := !bucket.MarkedForDeletion.Bool && 
		(!bucket.AccessError.Valid || bucket.AccessError.String == "" || 
		 (bucket.LastAccessibleAt.Valid && bucket.LastAccessibleAt.Time.After(time.Now().Add(-24*time.Hour))))
	
	var lastAccessibleAt *time.Time
	if bucket.LastAccessibleAt.Valid {
		lastAccessibleAt = &bucket.LastAccessibleAt.Time
	}
	
	accessError := ""
	if bucket.AccessError.Valid {
		accessError = bucket.AccessError.String
	}
	
	if scanStatus == "" {
		scanStatus = "never_scanned"
	}
	
	// Handle nullable CreatedAt
	creationDate := time.Time{}
	if bucket.CreatedAt.Valid {
		creationDate = bucket.CreatedAt.Time
	}
	
	region := ""
	if bucket.Region.Valid {
		region = bucket.Region.String
	}
	
	return dto.Bucket{
		Name:                bucket.Name,
		Region:              region,
		CreationDate:        creationDate,
		IsAccessible:        isAccessible,
		LastAccessibleAt:    lastAccessibleAt,
		AccessError:         accessError,
		ScanStatus:          scanStatus,
		LastScanError:       scanError,
		LastScanCompletedAt: scanCompletedAt,
	}
}

// convertToDTO converts database objects to DTO objects.
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
		result[i].Name = s.extractObjectName(obj.Key, obj.IsFolder.Bool)
	}
	return result
}

// formatSize formats file size in human readable format.
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

// extractObjectName extracts the display name from an object key.
func (s *Service) extractObjectName(key string, isFolder bool) string {
	if isFolder {
		return s.extractFolderName(key)
	}
	return s.extractFileName(key)
}

// extractFolderName extracts the folder name from a folder key.
func (s *Service) extractFolderName(key string) string {
	folderKey := strings.TrimSuffix(key, "/")
	if idx := strings.LastIndex(folderKey, "/"); idx != -1 {
		return folderKey[idx+1:]
	}
	return folderKey
}

// extractFileName extracts the file name from a file key.
func (s *Service) extractFileName(key string) string {
	if idx := strings.LastIndex(key, "/"); idx != -1 {
		return key[idx+1:]
	}
	return key
}