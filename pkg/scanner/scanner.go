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
			_, updateErr := s.queries.UpdateScanJobStatus(ctx, database.UpdateScanJobStatusParams{
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

		// Process each object
		for _, obj := range page.Contents {
			if err := s.processObject(ctx, bucket.ID, obj); err != nil {
				s.log.Error("Failed to process object", 
					slog.String("key", aws.ToString(obj.Key)), 
					slog.String("error", err.Error()))
				continue
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
			if err := s.processFolder(ctx, bucket.ID, aws.ToString(prefix.Prefix)); err != nil {
				s.log.Error("Failed to process folder", 
					slog.String("prefix", aws.ToString(prefix.Prefix)), 
					slog.String("error", err.Error()))
				continue
			}
		}
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
		slog.Int("objects", objectCount))

	return nil
}

// processObject processes a single S3 object and saves it to the database
func (s *Service) processObject(ctx context.Context, bucketID int32, obj types.Object) error {
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

	// Create or update the object
	_, err := s.queries.CreateS3Object(ctx, database.CreateS3ObjectParams{
		BucketID:     bucketID,
		Key:          key,
		Size:         *size,
		LastModified: sql.NullTime{Time: *lastModified, Valid: lastModified != nil},
		Etag:         sql.NullString{String: etag, Valid: etag != ""},
		StorageClass: sql.NullString{String: storageClass, Valid: storageClass != ""},
		IsFolder:     sql.NullBool{Bool: false, Valid: true},
		Prefix:       sql.NullString{String: prefix, Valid: prefix != ""},
	})

	return err
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
func (s *Service) processFolder(ctx context.Context, bucketID int32, folderPrefix string) error {
	// Remove trailing slash for folder name
	folderKey := strings.TrimSuffix(folderPrefix, "/")
	
	// Determine parent prefix
	parentPrefix := ""
	if idx := strings.LastIndex(folderKey, "/"); idx != -1 {
		parentPrefix = folderKey[:idx+1]
	}

	// Create the folder entry
	_, err := s.queries.CreateS3Object(ctx, database.CreateS3ObjectParams{
		BucketID:     bucketID,
		Key:          folderPrefix, // Keep trailing slash for folders
		Size:         0,
		LastModified: sql.NullTime{Time: time.Now(), Valid: true},
		Etag:         sql.NullString{},
		StorageClass: sql.NullString{},
		IsFolder:     sql.NullBool{Bool: true, Valid: true},
		Prefix:       sql.NullString{String: parentPrefix, Valid: parentPrefix != ""},
	})

	return err
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