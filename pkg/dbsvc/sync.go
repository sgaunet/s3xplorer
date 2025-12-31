package dbsvc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/sgaunet/s3xplorer/pkg/database"
)

var (
	// ErrPartialDeletionSync indicates that not all deletions were synced to the database.
	ErrPartialDeletionSync = errors.New("partial deletion sync failure")
)

// SyncUploadedObject creates or updates an S3 object record in the database after upload.
// This keeps the database in sync with S3 after a successful upload operation.
func (s *Service) SyncUploadedObject(
	ctx context.Context,
	bucketName, key string,
	size int64,
	etag, storageClass string,
) error {
	// Get bucket ID
	bucket, err := s.queries.GetBucket(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("bucket not found: %w", err)
	}

	// Determine if this is a folder (ends with /)
	isFolder := len(key) > 0 && key[len(key)-1] == '/'

	// Extract prefix (parent folder path)
	prefix := extractPrefix(key)

	// Create or update the object in database
	_, err = s.queries.CreateS3Object(ctx, database.CreateS3ObjectParams{
		BucketID:     bucket.ID,
		Key:          key,
		Size:         size,
		LastModified: sql.NullTime{Time: time.Now(), Valid: true},
		Etag:         sql.NullString{String: etag, Valid: etag != ""},
		StorageClass: sql.NullString{String: storageClass, Valid: storageClass != ""},
		IsFolder:     sql.NullBool{Bool: isFolder, Valid: true},
		Prefix:       sql.NullString{String: prefix, Valid: true},
	})

	if err != nil {
		return fmt.Errorf("failed to sync uploaded object: %w", err)
	}

	s.log.Debug("Synced uploaded object to database",
		slog.String("bucket", bucketName),
		slog.String("key", key))

	return nil
}

// SyncDeletedObject removes an S3 object record from the database after deletion.
// This keeps the database in sync with S3 after a successful delete operation.
func (s *Service) SyncDeletedObject(ctx context.Context, bucketName, key string) error {
	// Get bucket ID
	bucket, err := s.queries.GetBucket(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("bucket not found: %w", err)
	}

	// Delete the object from database
	err = s.queries.DeleteS3Object(ctx, database.DeleteS3ObjectParams{
		BucketID: bucket.ID,
		Key:      key,
	})

	if err != nil {
		return fmt.Errorf("failed to sync deleted object: %w", err)
	}

	s.log.Debug("Synced deleted object to database",
		slog.String("bucket", bucketName),
		slog.String("key", key))

	return nil
}

// SyncDeletedObjects removes multiple S3 object records from the database after bulk deletion.
func (s *Service) SyncDeletedObjects(ctx context.Context, bucketName string, keys []string) error {
	// Get bucket ID
	bucket, err := s.queries.GetBucket(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("bucket not found: %w", err)
	}

	// Delete each object (sqlc doesn't support bulk deletes easily, so we iterate)
	successCount := 0
	for _, key := range keys {
		err = s.queries.DeleteS3Object(ctx, database.DeleteS3ObjectParams{
			BucketID: bucket.ID,
			Key:      key,
		})
		if err != nil {
			s.log.Error("Failed to sync deleted object",
				slog.String("bucket", bucketName),
				slog.String("key", key),
				slog.String("error", err.Error()))
			// Continue deleting others
			continue
		}
		successCount++
	}

	if successCount != len(keys) {
		return fmt.Errorf("%w: synced %d of %d deleted objects", ErrPartialDeletionSync, successCount, len(keys))
	}

	s.log.Debug("Synced deleted objects to database",
		slog.String("bucket", bucketName),
		slog.Int("count", successCount))

	return nil
}

// extractPrefix extracts the parent folder path from a key.
// Examples:
//   - "folder/" -> ""
//   - "folder/file.txt" -> "folder/"
//   - "a/b/c/file.txt" -> "a/b/c/"
func extractPrefix(key string) string {
	// Remove trailing slash if present (for folders)
	k := key
	if len(k) > 0 && k[len(k)-1] == '/' {
		k = k[:len(k)-1]
	}

	// Find last slash
	lastSlash := -1
	for i := len(k) - 1; i >= 0; i-- {
		if k[i] == '/' {
			lastSlash = i
			break
		}
	}

	if lastSlash == -1 {
		return "" // Root level
	}

	return k[:lastSlash+1] // Include the trailing slash
}
