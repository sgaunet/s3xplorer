package scanner

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/sgaunet/s3xplorer/pkg/config"
	"github.com/sgaunet/s3xplorer/pkg/database"
)

// Service handles S3 bucket scanning operations
type Service struct {
	s3Client *s3.Client
	db       *sql.DB
	queries  *database.Queries
	cfg      config.Config
	log      *slog.Logger
}

// NewService creates a new scanner service
func NewService(cfg config.Config, s3Client *s3.Client, db *sql.DB) *Service {
	return &Service{
		s3Client: s3Client,
		db:       db,
		queries:  database.New(db),
		cfg:      cfg,
		log:      slog.New(slog.DiscardHandler),
	}
}

// SetLogger sets the logger for the scanner
func (s *Service) SetLogger(log *slog.Logger) {
	s.log = log
}

// ScanBucket scans an entire S3 bucket and saves objects to PostgreSQL
func (s *Service) ScanBucket(ctx context.Context, bucketName string) error {
	s.log.Info("Starting bucket scan", slog.String("bucket", bucketName))

	// Create or get bucket record
	bucket, err := s.queries.CreateBucket(ctx, database.CreateBucketParams{
		Name:   bucketName,
		Region: sql.NullString{String: s.cfg.S3Region, Valid: s.cfg.S3Region != ""},
	})
	if err != nil {
		return fmt.Errorf("failed to create/get bucket: %w", err)
	}

	// Create scan job
	scanJob, err := s.queries.CreateScanJob(ctx, database.CreateScanJobParams{
		BucketID: bucket.ID,
		Status:   "running",
	})
	if err != nil {
		return fmt.Errorf("failed to create scan job: %w", err)
	}

	// Update scan job to running
	_, err = s.queries.UpdateScanJobStatus(ctx, database.UpdateScanJobStatusParams{
		ID:      scanJob.ID,
		Column2: "running",
	})
	if err != nil {
		s.log.Error("Failed to update scan job status", slog.String("error", err.Error()))
	}

	objectCount := 0
	var scanErr error

	// Phase 1: Mark all existing objects as potentially deleted (if deletion sync is enabled)
	if s.cfg.EnableDeletionSync {
		s.log.Info("Phase 1: Marking all objects for deletion check", slog.String("bucket", bucketName))
		if err := s.queries.MarkAllObjectsForDeletion(ctx, bucket.ID); err != nil {
			scanErr = fmt.Errorf("failed to mark objects for deletion: %w", err)
			return scanErr
		}
	} else {
		s.log.Info("Deletion sync disabled - skipping Phase 1", slog.String("bucket", bucketName))
	}

	// Initialize counters for tracking scan statistics
	objectsCreated := 0
	objectsUpdated := 0
	objectsDeleted := 0

	// Scan the bucket
	defer func() {
		if scanErr != nil {
			_, updateErr := s.queries.UpdateScanJobError(ctx, database.UpdateScanJobErrorParams{
				ID:           scanJob.ID,
				ErrorMessage: sql.NullString{String: scanErr.Error(), Valid: true},
			})
			if updateErr != nil {
				s.log.Error("Failed to update scan job error", slog.String("error", updateErr.Error()))
			}
		} else {
			// Update final statistics
			_, updateErr := s.queries.UpdateScanJobStats(ctx, database.UpdateScanJobStatsParams{
				ID:             scanJob.ID,
				ObjectsScanned: sql.NullInt32{Int32: int32(objectCount), Valid: true},
				ObjectsCreated: sql.NullInt32{Int32: int32(objectsCreated), Valid: true},
				ObjectsUpdated: sql.NullInt32{Int32: int32(objectsUpdated), Valid: true},
				ObjectsDeleted: sql.NullInt32{Int32: int32(objectsDeleted), Valid: true},
			})
			if updateErr != nil {
				s.log.Error("Failed to update scan job stats", slog.String("error", updateErr.Error()))
			}

			_, updateErr = s.queries.UpdateScanJobStatus(ctx, database.UpdateScanJobStatusParams{
				ID:      scanJob.ID,
				Column2: "completed",
			})
			if updateErr != nil {
				s.log.Error("Failed to update scan job status", slog.String("error", updateErr.Error()))
			}
		}
	}()

	// Use ListObjectsV2 to get all objects
	paginator := s3.NewListObjectsV2Paginator(s.s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(s.cfg.Prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			scanErr = fmt.Errorf("failed to list objects: %w", err)
			return scanErr
		}

		// Process each object (Phase 2: Update/create and unmark for deletion)
		for _, obj := range page.Contents {
			isNew, err := s.processObject(ctx, bucket.ID, obj)
			if err != nil {
				s.log.Error("Failed to process object", 
					slog.String("key", aws.ToString(obj.Key)), 
					slog.String("error", err.Error()))
				continue
			}

			// Track creation vs update statistics
			if isNew {
				objectsCreated++
			} else {
				objectsUpdated++
			}
			objectCount++

			// Update progress every 100 objects
			if objectCount%100 == 0 {
				_, err := s.queries.UpdateScanJobProgress(ctx, database.UpdateScanJobProgressParams{
					ID:             scanJob.ID,
					ObjectsScanned: sql.NullInt32{Int32: int32(objectCount), Valid: true},
				})
				if err != nil {
					s.log.Error("Failed to update scan job progress", slog.String("error", err.Error()))
				}
			}
		}

		// Process common prefixes (folders)
		for _, prefix := range page.CommonPrefixes {
			isNew, err := s.processFolder(ctx, bucket.ID, aws.ToString(prefix.Prefix))
			if err != nil {
				s.log.Error("Failed to process folder", 
					slog.String("prefix", aws.ToString(prefix.Prefix)), 
					slog.String("error", err.Error()))
				continue
			}

			// Track folder creation vs update statistics
			if isNew {
				objectsCreated++
			} else {
				objectsUpdated++
			}
		}
	}

	// Phase 3: Delete objects that are still marked for deletion (if deletion sync is enabled)
	if s.cfg.EnableDeletionSync {
		s.log.Info("Phase 3: Cleaning up deleted objects", slog.String("bucket", bucketName))
		markedCount, err := s.queries.CountMarkedObjects(ctx, bucket.ID)
		if err != nil {
			s.log.Error("Failed to count marked objects", slog.String("error", err.Error()))
		} else {
			objectsDeleted = int(markedCount)
			if objectsDeleted > 0 {
				s.log.Info("Deleting objects no longer in S3", 
					slog.String("bucket", bucketName), 
					slog.Int("count", objectsDeleted))
				if err := s.queries.DeleteMarkedObjects(ctx, bucket.ID); err != nil {
					s.log.Error("Failed to delete marked objects", slog.String("error", err.Error()))
					// Don't fail the entire scan if deletion cleanup fails
					objectsDeleted = 0
				}
			}
		}
	} else {
		s.log.Info("Deletion sync disabled - skipping Phase 3", slog.String("bucket", bucketName))
	}

	// Final progress update
	_, err = s.queries.UpdateScanJobProgress(ctx, database.UpdateScanJobProgressParams{
		ID:             scanJob.ID,
		ObjectsScanned: sql.NullInt32{Int32: int32(objectCount), Valid: true},
	})
	if err != nil {
		s.log.Error("Failed to update final scan job progress", slog.String("error", err.Error()))
	}

	s.log.Info("Bucket scan completed", 
		slog.String("bucket", bucketName), 
		slog.Int("objects_scanned", objectCount),
		slog.Int("objects_created", objectsCreated),
		slog.Int("objects_updated", objectsUpdated),
		slog.Int("objects_deleted", objectsDeleted))

	return nil
}

// processObject processes a single S3 object and saves it to the database
// Returns true if object was newly created, false if it was updated
func (s *Service) processObject(ctx context.Context, bucketID int32, obj types.Object) (bool, error) {
	key := aws.ToString(obj.Key)
	size := obj.Size
	lastModified := obj.LastModified
	etag := aws.ToString(obj.ETag)
	storageClass := string(obj.StorageClass)

	// Determine prefix (folder path)
	prefix := ""
	if idx := strings.LastIndex(key, "/"); idx != -1 {
		prefix = key[:idx+1]
	}

	// Create missing intermediate folder entries
	if prefix != "" {
		if err := s.ensureParentFolders(ctx, bucketID, prefix); err != nil {
			s.log.Error("Failed to create parent folders", 
				slog.String("prefix", prefix), 
				slog.String("error", err.Error()))
		}
	}

	// Check if object already exists to determine if it's new or updated
	_, err := s.queries.GetS3Object(ctx, database.GetS3ObjectParams{
		BucketID: bucketID,
		Key:      key,
	})
	isNew := err != nil // If we get an error, the object doesn't exist

	// Create or update the object
	_, err = s.queries.CreateS3Object(ctx, database.CreateS3ObjectParams{
		BucketID:     bucketID,
		Key:          key,
		Size:         *size,
		LastModified: sql.NullTime{Time: *lastModified, Valid: lastModified != nil},
		Etag:         sql.NullString{String: etag, Valid: etag != ""},
		StorageClass: sql.NullString{String: storageClass, Valid: storageClass != ""},
		IsFolder:     sql.NullBool{Bool: false, Valid: true},
		Prefix:       sql.NullString{String: prefix, Valid: prefix != ""},
	})
	if err != nil {
		return false, err
	}

	// Unmark the object for deletion since we found it in S3 (if deletion sync is enabled)
	if s.cfg.EnableDeletionSync {
		if err := s.queries.UnmarkObjectForDeletion(ctx, database.UnmarkObjectForDeletionParams{
			BucketID: bucketID,
			Key:      key,
		}); err != nil {
			s.log.Error("Failed to unmark object for deletion", 
				slog.String("key", key), 
				slog.String("error", err.Error()))
			// Don't fail the scan if unmarking fails
		}
	}

	return isNew, nil
}

// ensureParentFolders creates all missing intermediate folder entries for a given path
func (s *Service) ensureParentFolders(ctx context.Context, bucketID int32, fullPath string) error {
	// Remove trailing slash and split the path
	cleanPath := strings.TrimSuffix(fullPath, "/")
	if cleanPath == "" {
		return nil
	}
	
	parts := strings.Split(cleanPath, "/")
	currentPath := ""
	
	// Create each folder level
	for i, part := range parts {
		if part == "" {
			continue
		}
		
		// Build the current folder path
		if currentPath != "" {
			currentPath += "/"
		}
		currentPath += part
		folderKey := currentPath + "/"
		
		// Determine the parent prefix for this folder
		parentPrefix := ""
		if i > 0 {
			// Parent is all parts before this one
			parentPath := strings.Join(parts[:i], "/")
			if parentPath != "" {
				parentPrefix = parentPath + "/"
			}
		}
		
		// Create the folder entry (this will ignore if it already exists due to ON CONFLICT)
		_, err := s.queries.CreateS3Object(ctx, database.CreateS3ObjectParams{
			BucketID:     bucketID,
			Key:          folderKey,
			Size:         0,
			LastModified: sql.NullTime{Time: time.Now(), Valid: true},
			Etag:         sql.NullString{},
			StorageClass: sql.NullString{},
			IsFolder:     sql.NullBool{Bool: true, Valid: true},
			Prefix:       sql.NullString{String: parentPrefix, Valid: parentPrefix != ""},
		})
		if err != nil {
			return fmt.Errorf("failed to create folder %s: %w", folderKey, err)
		}
	}
	
	return nil
}

// processFolder processes a folder prefix and saves it to the database
// Returns true if folder was newly created, false if it was updated
func (s *Service) processFolder(ctx context.Context, bucketID int32, folderPrefix string) (bool, error) {
	// Remove trailing slash for folder name
	folderKey := strings.TrimSuffix(folderPrefix, "/")
	
	// Determine parent prefix
	parentPrefix := ""
	if idx := strings.LastIndex(folderKey, "/"); idx != -1 {
		parentPrefix = folderKey[:idx+1]
	}

	// Check if folder already exists to determine if it's new or updated
	_, err := s.queries.GetS3Object(ctx, database.GetS3ObjectParams{
		BucketID: bucketID,
		Key:      folderPrefix,
	})
	isNew := err != nil // If we get an error, the folder doesn't exist

	// Create the folder entry
	_, err = s.queries.CreateS3Object(ctx, database.CreateS3ObjectParams{
		BucketID:     bucketID,
		Key:          folderPrefix, // Keep trailing slash for folders
		Size:         0,
		LastModified: sql.NullTime{Time: time.Now(), Valid: true},
		Etag:         sql.NullString{},
		StorageClass: sql.NullString{},
		IsFolder:     sql.NullBool{Bool: true, Valid: true},
		Prefix:       sql.NullString{String: parentPrefix, Valid: parentPrefix != ""},
	})
	if err != nil {
		return false, err
	}

	// Unmark the folder for deletion since we found it in S3 (if deletion sync is enabled)
	if s.cfg.EnableDeletionSync {
		if err := s.queries.UnmarkObjectForDeletion(ctx, database.UnmarkObjectForDeletionParams{
			BucketID: bucketID,
			Key:      folderPrefix,
		}); err != nil {
			s.log.Error("Failed to unmark folder for deletion", 
				slog.String("key", folderPrefix), 
				slog.String("error", err.Error()))
			// Don't fail the scan if unmarking fails
		}
	}

	return isNew, nil
}

// GetScanStatus returns the status of the latest scan job for a bucket
func (s *Service) GetScanStatus(ctx context.Context, bucketName string) (*database.ScanJob, error) {
	bucket, err := s.queries.GetBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %w", err)
	}

	scanJob, err := s.queries.GetLatestScanJob(ctx, bucket.ID)
	if err != nil {
		return nil, fmt.Errorf("no scan jobs found: %w", err)
	}

	return &scanJob, nil
}

// DiscoverAndScanAllBuckets discovers all available buckets and scans them
func (s *Service) DiscoverAndScanAllBuckets(ctx context.Context) error {
	s.log.Info("Starting discovery and initial scan of all buckets")

	// If a specific bucket is configured, only scan that bucket
	if s.cfg.Bucket != "" {
		s.log.Info("Scanning configured bucket", slog.String("bucket", s.cfg.Bucket))
		return s.ScanBucket(ctx, s.cfg.Bucket)
	}

	// Discover all available buckets
	buckets, err := s.discoverBuckets(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover buckets: %w", err)
	}

	s.log.Info("Discovered buckets", slog.Int("count", len(buckets)))

	// Scan each bucket
	for _, bucket := range buckets {
		s.log.Info("Scanning bucket", slog.String("bucket", bucket))
		if err := s.ScanBucket(ctx, bucket); err != nil {
			s.log.Error("Failed to scan bucket", 
				slog.String("bucket", bucket), 
				slog.String("error", err.Error()))
			continue // Continue with other buckets
		}
	}

	s.log.Info("Completed initial scan of all buckets")
	return nil
}

// discoverBuckets lists all available S3 buckets
func (s *Service) discoverBuckets(ctx context.Context) ([]string, error) {
	s.log.Debug("Discovering available buckets")

	result, err := s.s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to list buckets: %w", err)
	}

	buckets := make([]string, 0, len(result.Buckets))
	for _, bucket := range result.Buckets {
		bucketName := aws.ToString(bucket.Name)
		buckets = append(buckets, bucketName)
		s.log.Debug("Found bucket", slog.String("name", bucketName))
	}

	return buckets, nil
}

// ScanConfiguredBucket scans only the bucket specified in configuration
func (s *Service) ScanConfiguredBucket(ctx context.Context) error {
	if s.cfg.Bucket == "" {
		return fmt.Errorf("no bucket configured for scanning")
	}

	s.log.Info("Scanning configured bucket", slog.String("bucket", s.cfg.Bucket))
	return s.ScanBucket(ctx, s.cfg.Bucket)
}