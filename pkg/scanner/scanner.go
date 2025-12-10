// Package scanner provides S3 bucket scanning and synchronization functionality.
package scanner

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/sgaunet/s3xplorer/pkg/config"
	"github.com/sgaunet/s3xplorer/pkg/database"
)

// ErrNoBucketConfigured is returned when no bucket is configured for scanning.
var ErrNoBucketConfigured = errors.New("no bucket configured for scanning")

// Service handles S3 bucket scanning operations.
type Service struct {
	s3Client *s3.Client
	db       *sql.DB
	queries  *database.Queries
	cfg      config.Config
	log      *slog.Logger
}

// BucketErrorType represents the type of bucket access error.
type BucketErrorType string

const (
	// ErrorTypeNotFound indicates the bucket does not exist (404).
	ErrorTypeNotFound BucketErrorType = "not_found"
	// ErrorTypeAccessDenied indicates access is denied (403).
	ErrorTypeAccessDenied BucketErrorType = "access_denied"
	// ErrorTypeTemporary indicates a temporary error (5xx, network issues).
	ErrorTypeTemporary BucketErrorType = "temporary"
	// ErrorTypeUnknown indicates an unknown error type.
	ErrorTypeUnknown BucketErrorType = "unknown"
)

// NewService creates a new scanner service.
func NewService(cfg config.Config, s3Client *s3.Client, db *sql.DB) *Service {
	return &Service{
		s3Client: s3Client,
		db:       db,
		queries:  database.New(db),
		cfg:      cfg,
		log:      slog.New(slog.DiscardHandler),
	}
}

// SetLogger sets the logger for the scanner.
func (s *Service) SetLogger(log *slog.Logger) {
	s.log = log
}

// classifyAPIError classifies AWS API errors.
func classifyAPIError(errorCode string) BucketErrorType {
	switch errorCode {
	case "NoSuchBucket", "BucketNotFound":
		return ErrorTypeNotFound
	case "AccessDenied", "Forbidden":
		return ErrorTypeAccessDenied
	case "InternalError", "ServiceUnavailable", "SlowDown":
		return ErrorTypeTemporary
	default:
		return ErrorTypeUnknown
	}
}

// classifyHTTPError classifies HTTP status code errors.
func classifyHTTPError(statusCode int) BucketErrorType {
	switch statusCode {
	case http.StatusNotFound:
		return ErrorTypeNotFound
	case http.StatusForbidden:
		return ErrorTypeAccessDenied
	case http.StatusInternalServerError, http.StatusBadGateway,
		http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return ErrorTypeTemporary
	default:
		return ErrorTypeUnknown
	}
}

// isNetworkError checks if an error is network-related.
func isNetworkError(err error) bool {
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "network")
}

// ScanBucket scans an entire S3 bucket and saves objects to PostgreSQL.
func (s *Service) ScanBucket(ctx context.Context, bucketName string) error {
	s.log.Info("Starting bucket scan", slog.String("bucket", bucketName))

	// First, validate bucket accessibility before proceeding (unless skipped)
	if err := s.performBucketValidation(ctx, bucketName); err != nil {
		return err
	}

	// Create or get bucket record and initialize scan job
	bucket, scanJob, err := s.initializeBucketAndScanJob(ctx, bucketName)
	if err != nil {
		return err
	}

	objectCount := 0
	var scanErr error

	// Phase 1: Mark all existing objects as potentially deleted (if deletion sync is enabled)
	if err := s.performDeletionSyncPhase(ctx, bucketName, bucket.ID); err != nil {
		return err
	}

	// Initialize counters for tracking scan statistics
	objectsCreated := 0
	objectsUpdated := 0
	objectsDeleted := 0

	// Scan the bucket
	defer s.finalizeScanJob(
		ctx, bucketName, scanJob.ID, &objectCount, &objectsCreated, &objectsUpdated, &objectsDeleted, &scanErr,
	)

	// Phase 2: Scan and process all S3 objects and folders
	scanErr = s.performS3ObjectScan(ctx, bucketName, bucket.ID, scanJob.ID, &objectCount, &objectsCreated, &objectsUpdated)

	// Phase 3: Delete objects that are still marked for deletion (if deletion sync is enabled)
	objectsDeleted = s.performDeletionCleanup(ctx, bucketName, bucket.ID)

	// Final progress update
	_, err = s.queries.UpdateScanJobProgress(ctx, database.UpdateScanJobProgressParams{
		ID:             scanJob.ID,
		ObjectsScanned: sql.NullInt32{Int32: int32(min(objectCount, math.MaxInt32)), Valid: true},
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


// GetScanStatus returns the status of the latest scan job for a bucket.
func (s *Service) GetScanStatus(ctx context.Context, bucketName string) (*database.ScanJob, error) {
	bucket, err := s.queries.GetBucket(ctx, bucketName)
	if err != nil {
		return nil, fmt.Errorf("bucket not found: %w", err)
	}

	scanJob, err := s.queries.GetLatestScanJob(ctx, sql.NullInt32{Int32: bucket.ID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("no scan jobs found: %w", err)
	}

	return &scanJob, nil
}

// DiscoverAndScanAllBuckets discovers all available buckets, validates them, and scans them.
func (s *Service) DiscoverAndScanAllBuckets(ctx context.Context) error {
	s.log.Info("Starting discovery and initial scan of all buckets")

	// If a specific bucket is configured, only scan that bucket
	if s.cfg.S3.Bucket != "" {
		s.log.Info("Scanning configured bucket", slog.String("bucket", s.cfg.S3.Bucket))

		// Perform bucket validation if enabled
		if s.cfg.BucketSync.Enable {
			_, _, _, _, err := s.validateAndSyncBuckets(ctx, []string{s.cfg.S3.Bucket})
			if err != nil {
				s.log.Error("Failed to validate configured bucket",
					slog.String("bucket", s.cfg.S3.Bucket),
					slog.String("error", err.Error()))
				// Continue with scan even if validation fails
			}
		}

		return s.ScanBucket(ctx, s.cfg.S3.Bucket)
	}

	// Discover all available buckets
	buckets, err := s.discoverBuckets(ctx)
	if err != nil {
		return fmt.Errorf("failed to discover buckets: %w", err)
	}

	s.log.Info("Discovered buckets", slog.Int("count", len(buckets)))

	// Perform bucket validation and synchronization
	bucketsValidated, bucketsMarkedInaccessible, bucketsCleanedUp, bucketValidationErrors, err :=
		s.validateAndSyncBuckets(ctx, buckets)
	if err != nil {
		s.log.Error("Failed to validate and sync buckets", slog.String("error", err.Error()))
		return fmt.Errorf("failed to validate buckets: %w", err)
	}

	s.log.Info("Bucket validation completed",
		slog.Int("buckets_validated", bucketsValidated),
		slog.Int("buckets_marked_inaccessible", bucketsMarkedInaccessible),
		slog.Int("buckets_cleaned_up", bucketsCleanedUp),
		slog.Int("bucket_validation_errors", bucketValidationErrors))

	// Scan each discovered bucket using the tracking function
	if err := s.ScanAllBucketsWithTracking(ctx, buckets, bucketsValidated,
		bucketsMarkedInaccessible, bucketsCleanedUp, bucketValidationErrors); err != nil {
		s.log.Error("Failed to scan buckets with tracking", slog.String("error", err.Error()))
		return fmt.Errorf("failed to scan buckets: %w", err)
	}

	s.log.Info("Completed discovery, validation, and scan of all buckets",
		slog.Int("buckets_discovered", len(buckets)),
		slog.Int("buckets_validated", bucketsValidated))
	return nil
}



// ScanAllBucketsWithTracking scans multiple buckets and tracks bucket sync statistics.
func (s *Service) ScanAllBucketsWithTracking(
	ctx context.Context, buckets []string, bucketsValidated, _, _, _ int,
) error {
	if len(buckets) == 0 {
		s.log.Info("No buckets to scan")
		return nil
	}

	s.log.Info("Starting bulk bucket scan with tracking",
		slog.Int("bucket_count", len(buckets)),
		slog.Int("buckets_validated", bucketsValidated))

	stats := s.scanBucketsAndCollectStats(ctx, buckets)
	s.logBulkScanResults(stats)
	return nil
}

// bucketScanStats holds statistics for bulk bucket scanning.
type bucketScanStats struct {
	totalObjectsScanned      int
	totalObjectsCreated      int
	totalObjectsUpdated      int
	totalObjectsDeleted      int
	bucketsScannedSuccessfully int
	bucketsFailedPermanently   int
	bucketsFailedTemporarily   int
}


// ScanConfiguredBucket scans only the bucket specified in configuration.
func (s *Service) ScanConfiguredBucket(ctx context.Context) error {
	if s.cfg.S3.Bucket == "" {
		return ErrNoBucketConfigured
	}

	s.log.Info("Scanning configured bucket", slog.String("bucket", s.cfg.S3.Bucket))
	return s.ScanBucket(ctx, s.cfg.S3.Bucket)
}

// validateAndSyncBuckets performs bucket-level validation and synchronization.
func (s *Service) validateAndSyncBuckets(ctx context.Context, discoveredBuckets []string) (int, int, int, int, error) {
	if !s.cfg.BucketSync.Enable {
		s.log.Debug("Bucket sync disabled - skipping bucket validation")
		return 0, 0, 0, 0, nil
	}

	s.log.Info("Starting bucket validation and synchronization")

	// Phase 1: Mark all existing buckets for deletion validation
	if err := s.performPhase1BucketMarking(ctx); err != nil {
		return 0, 0, 0, 0, err
	}

	// Phase 2: Validate discovered buckets
	bucketsValidated, bucketsMarkedInaccessible, bucketValidationErrors := s.performPhase2BucketValidation(
		ctx, discoveredBuckets)

	// Phase 3: Clean up long-term inaccessible buckets
	bucketsCleanedUp := s.performPhase3BucketCleanup(ctx)

	s.log.Info("Bucket validation and synchronization completed",
		slog.Int("buckets_validated", bucketsValidated),
		slog.Int("buckets_marked_inaccessible", bucketsMarkedInaccessible),
		slog.Int("buckets_cleaned_up", bucketsCleanedUp),
		slog.Int("bucket_validation_errors", bucketValidationErrors))

	return bucketsValidated, bucketsMarkedInaccessible, bucketsCleanedUp, bucketValidationErrors, nil
}

// ensureParentFolders creates all missing intermediate folder entries for a given path.
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

// processFolder processes a folder prefix and saves it to the database.
// Returns true if folder was newly created, false if it was updated.
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
		return false, fmt.Errorf("failed to create S3 folder: %w", err)
	}

	// Unmark the folder for deletion since we found it in S3 (if deletion sync is enabled)
	if s.cfg.Scan.EnableDeletionSync {
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

// discoverBuckets lists all available S3 buckets.
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

// validateBucketAccessibility tests if a bucket is accessible using HeadBucket operation.
func (s *Service) validateBucketAccessibility(ctx context.Context, bucketName string) error {
	s.log.Debug("Validating bucket accessibility", slog.String("bucket", bucketName))

	// Use HeadBucket to check if bucket is accessible
	_, err := s.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})

	if err != nil {
		errorType := s.classifyBucketError(err)

		// Use appropriate log level based on error type
		switch errorType {
		case ErrorTypeNotFound:
			s.log.Warn("Bucket not found",
				slog.String("bucket", bucketName),
				slog.String("error", err.Error()),
				slog.String("error_type", string(errorType)))
		case ErrorTypeAccessDenied:
			s.log.Warn("Bucket access denied",
				slog.String("bucket", bucketName),
				slog.String("error", err.Error()),
				slog.String("error_type", string(errorType)))
		case ErrorTypeTemporary:
			s.log.Debug("Bucket temporarily inaccessible",
				slog.String("bucket", bucketName),
				slog.String("error", err.Error()),
				slog.String("error_type", string(errorType)))
		case ErrorTypeUnknown:
			s.log.Error("Unknown bucket accessibility error",
				slog.String("bucket", bucketName),
				slog.String("error", err.Error()),
				slog.String("error_type", string(errorType)))
		}

		return fmt.Errorf("bucket %s is not accessible: %w", bucketName, err)
	}

	s.log.Debug("Bucket accessibility check passed", slog.String("bucket", bucketName))
	return nil
}

// scanBucketsAndCollectStats scans buckets and collects statistics.
func (s *Service) scanBucketsAndCollectStats(ctx context.Context, buckets []string) bucketScanStats {
	stats := bucketScanStats{}

	for _, bucket := range buckets {
		s.log.Info("Scanning bucket", slog.String("bucket", bucket))
		if err := s.ScanBucket(ctx, bucket); err != nil {
			s.handleBucketScanError(bucket, err, &stats)
			continue
		}
		stats.bucketsScannedSuccessfully++
		s.aggregateBucketStats(ctx, bucket, &stats)
	}
	return stats
}

// handleBucketScanError processes scan errors and updates statistics.
func (s *Service) handleBucketScanError(bucket string, err error, stats *bucketScanStats) {
	errorType := s.classifyBucketError(err)
	s.log.Error("Failed to scan bucket",
		slog.String("bucket", bucket),
		slog.String("error", err.Error()),
		slog.String("error_type", string(errorType)))

	if errorType == ErrorTypeNotFound || errorType == ErrorTypeAccessDenied {
		stats.bucketsFailedPermanently++
	} else {
		stats.bucketsFailedTemporarily++
	}
}

// aggregateBucketStats adds individual bucket statistics to the total.
func (s *Service) aggregateBucketStats(ctx context.Context, bucket string, stats *bucketScanStats) {
	bucketRecord, err := s.queries.GetBucket(ctx, bucket)
	if err != nil {
		s.log.Debug("Could not get bucket record for stats aggregation",
			slog.String("bucket", bucket))
		return
	}

	latestScanJob, err := s.queries.GetLatestScanJob(ctx, sql.NullInt32{Int32: bucketRecord.ID, Valid: true})
	if err != nil {
		s.log.Debug("Could not get latest scan job for stats aggregation",
			slog.String("bucket", bucket))
		return
	}

	// Aggregate statistics
	if latestScanJob.ObjectsScanned.Valid {
		stats.totalObjectsScanned += int(latestScanJob.ObjectsScanned.Int32)
	}
	if latestScanJob.ObjectsCreated.Valid {
		stats.totalObjectsCreated += int(latestScanJob.ObjectsCreated.Int32)
	}
	if latestScanJob.ObjectsUpdated.Valid {
		stats.totalObjectsUpdated += int(latestScanJob.ObjectsUpdated.Int32)
	}
	if latestScanJob.ObjectsDeleted.Valid {
		stats.totalObjectsDeleted += int(latestScanJob.ObjectsDeleted.Int32)
	}
}

// logBulkScanResults logs the final results of bulk bucket scanning.
func (s *Service) logBulkScanResults(stats bucketScanStats) {
	s.log.Info("Completed bulk bucket scan with tracking",
		slog.Int("buckets_scanned_successfully", stats.bucketsScannedSuccessfully),
		slog.Int("buckets_failed_permanently", stats.bucketsFailedPermanently),
		slog.Int("buckets_failed_temporarily", stats.bucketsFailedTemporarily),
		slog.Int("total_objects_scanned", stats.totalObjectsScanned),
		slog.Int("total_objects_created", stats.totalObjectsCreated),
		slog.Int("total_objects_updated", stats.totalObjectsUpdated),
		slog.Int("total_objects_deleted", stats.totalObjectsDeleted))
}

// processObject processes a single S3 object and saves it to the database
// Returns true if object was newly created, false if it was updated.
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
		return false, fmt.Errorf("failed to create S3 object: %w", err)
	}

	// Unmark the object for deletion since we found it in S3 (if deletion sync is enabled)
	if s.cfg.Scan.EnableDeletionSync {
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

// classifyBucketError classifies S3 bucket access errors by type.
func (s *Service) classifyBucketError(err error) BucketErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}

	// Check for AWS API errors
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return classifyAPIError(apiErr.ErrorCode())
	}

	// Check for HTTP response errors
	var httpErr *smithyhttp.ResponseError
	if errors.As(err, &httpErr) {
		return classifyHTTPError(httpErr.HTTPStatusCode())
	}

	// Check for network/connection errors
	if isNetworkError(err) {
		return ErrorTypeTemporary
	}

	return ErrorTypeUnknown
}

// formatErrorWithClassification formats an error with classification information.
func (s *Service) formatErrorWithClassification(err error, context string) string {
	if err == nil {
		return ""
	}

	errorType := s.classifyBucketError(err)
	return fmt.Sprintf("%s (%s): %s", context, errorType, err.Error())
}

// performBucketValidation validates bucket accessibility and handles errors.
func (s *Service) performBucketValidation(ctx context.Context, bucketName string) error {
	if s.cfg.S3.SkipBucketValidation {
		s.log.Info("Skipping bucket validation", slog.String("bucket", bucketName))
		return nil
	}

	if err := s.validateBucketAccessibility(ctx, bucketName); err != nil {
		errorType := s.classifyBucketError(err)
		s.log.Error("Bucket accessibility check failed",
			slog.String("bucket", bucketName),
			slog.String("error", err.Error()),
			slog.String("error_type", string(errorType)))

		// For permanent errors, mark the bucket as inaccessible in the database
		if errorType == ErrorTypeNotFound || errorType == ErrorTypeAccessDenied {
			return s.handlePermanentBucketError(ctx, bucketName, err, errorType)
		}

		return fmt.Errorf("bucket %s is not accessible (%s): %w", bucketName, errorType, err)
	}

	return nil
}

// handlePermanentBucketError handles permanent bucket access errors.
func (s *Service) handlePermanentBucketError(
	ctx context.Context, bucketName string, err error, errorType BucketErrorType,
) error {
	// Create or get bucket record to mark it as inaccessible
	bucket, bucketErr := s.queries.CreateBucket(ctx, database.CreateBucketParams{
		Name:   bucketName,
		Region: sql.NullString{String: s.cfg.S3.Region, Valid: s.cfg.S3.Region != ""},
	})
	if bucketErr == nil {
		// Mark bucket for deletion and update access error
		markErr := s.queries.MarkBucketForDeletion(ctx, bucket.ID)
		if markErr != nil {
			s.log.Error("Failed to mark bucket for deletion",
				slog.String("bucket", bucketName),
				slog.String("error", markErr.Error()))
		}

		updateErr := s.queries.UpdateBucketAccessError(ctx, database.UpdateBucketAccessErrorParams{
			ID:          bucket.ID,
			AccessError: sql.NullString{String: err.Error(), Valid: true},
		})
		if updateErr != nil {
			s.log.Error("Failed to update bucket access error",
				slog.String("bucket", bucketName),
				slog.String("error", updateErr.Error()))
		}

		s.log.Warn("Bucket marked as inaccessible due to permanent error",
			slog.String("bucket", bucketName),
			slog.String("error_type", string(errorType)))
	}

	return fmt.Errorf("bucket %s is not accessible (%s): %w", bucketName, errorType, err)
}

// initializeBucketAndScanJob creates bucket record and initializes scan job.
func (s *Service) initializeBucketAndScanJob(
	ctx context.Context, bucketName string,
) (*database.Bucket, *database.ScanJob, error) {
	// Create or get bucket record
	bucket, err := s.queries.CreateBucket(ctx, database.CreateBucketParams{
		Name:   bucketName,
		Region: sql.NullString{String: s.cfg.S3.Region, Valid: s.cfg.S3.Region != ""},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create/get bucket: %w", err)
	}

	// Since bucket is accessible, unmark it for deletion and clear any access errors
	if unmarkErr := s.queries.UnmarkBucketForDeletion(ctx, bucket.ID); unmarkErr != nil {
		s.log.Error("Failed to unmark bucket for deletion",
			slog.String("bucket", bucketName),
			slog.String("error", unmarkErr.Error()))
	}

	// Create scan job
	scanJob, err := s.queries.CreateScanJob(ctx, database.CreateScanJobParams{
		BucketID: sql.NullInt32{Int32: bucket.ID, Valid: true},
		Status:   "running",
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create scan job: %w", err)
	}

	// Update scan job to running
	_, err = s.queries.UpdateScanJobStatus(ctx, database.UpdateScanJobStatusParams{
		ID:      scanJob.ID,
		Column2: "running",
	})
	if err != nil {
		s.log.Error("Failed to update scan job status", slog.String("error", err.Error()))
	}

	return &bucket, &scanJob, nil
}

// performDeletionSyncPhase marks objects for deletion if deletion sync is enabled.
func (s *Service) performDeletionSyncPhase(ctx context.Context, bucketName string, bucketID int32) error {
	if s.cfg.Scan.EnableDeletionSync {
		s.log.Info("Phase 1: Marking all objects for deletion check", slog.String("bucket", bucketName))
		if err := s.queries.MarkAllObjectsForDeletion(ctx, bucketID); err != nil {
			return fmt.Errorf("failed to mark objects for deletion: %w", err)
		}
	} else {
		s.log.Info("Deletion sync disabled - skipping Phase 1", slog.String("bucket", bucketName))
	}
	return nil
}

// finalizeScanJob handles scan job completion and statistics updates.
func (s *Service) finalizeScanJob(
	ctx context.Context, _ string, scanJobID int32,
	objectCount, objectsCreated, objectsUpdated, objectsDeleted *int,
	scanErr *error,
) {
	if *scanErr != nil {
		// Format error with classification for better tracking
		errorMsg := s.formatErrorWithClassification(*scanErr, "Bucket scan failed")
		_, updateErr := s.queries.UpdateScanJobError(ctx, database.UpdateScanJobErrorParams{
			ID:           scanJobID,
			ErrorMessage: sql.NullString{String: errorMsg, Valid: true},
		})
		if updateErr != nil {
			s.log.Error("Failed to update scan job error", slog.String("error", updateErr.Error()))
		}
	} else {
		// Update final statistics including bucket sync stats (default to 0 for individual bucket scans)
		_, updateErr := s.queries.UpdateScanJobFullStats(ctx, database.UpdateScanJobFullStatsParams{
			ID:                        scanJobID,
			ObjectsScanned:            sql.NullInt32{Int32: int32(min(*objectCount, math.MaxInt32)), Valid: true},
			ObjectsCreated:            sql.NullInt32{Int32: int32(*objectsCreated), Valid: true},
			ObjectsUpdated:            sql.NullInt32{Int32: int32(*objectsUpdated), Valid: true},
			ObjectsDeleted:            sql.NullInt32{Int32: int32(*objectsDeleted), Valid: true},
			BucketsValidated:          sql.NullInt32{Int32: 0, Valid: true}, // Individual bucket scans don't validate buckets
			BucketsMarkedInaccessible: sql.NullInt32{Int32: 0, Valid: true},
			BucketsCleanedUp:          sql.NullInt32{Int32: 0, Valid: true},
			BucketValidationErrors:    sql.NullInt32{Int32: 0, Valid: true},
		})
		if updateErr != nil {
			s.log.Error("Failed to update scan job stats", slog.String("error", updateErr.Error()))
		}

		_, updateErr = s.queries.UpdateScanJobStatus(ctx, database.UpdateScanJobStatusParams{
			ID:      scanJobID,
			Column2: "completed",
		})
		if updateErr != nil {
			s.log.Error("Failed to update scan job status", slog.String("error", updateErr.Error()))
		}
	}
}

// performDeletionCleanup handles the deletion of objects marked for removal.
func (s *Service) performDeletionCleanup(ctx context.Context, bucketName string, bucketID int32) int {
	if !s.cfg.Scan.EnableDeletionSync {
		s.log.Info("Deletion sync disabled - skipping Phase 3", slog.String("bucket", bucketName))
		return 0
	}

	s.log.Info("Phase 3: Cleaning up deleted objects", slog.String("bucket", bucketName))
	markedCount, err := s.queries.CountMarkedObjects(ctx, bucketID)
	if err != nil {
		s.log.Error("Failed to count marked objects", slog.String("error", err.Error()))
		return 0
	}

	objectsDeleted := int(markedCount)
	if objectsDeleted > 0 {
		s.log.Info("Deleting objects no longer in S3",
			slog.String("bucket", bucketName),
			slog.Int("count", objectsDeleted))
		if err := s.queries.DeleteMarkedObjects(ctx, bucketID); err != nil {
			s.log.Error("Failed to delete marked objects", slog.String("error", err.Error()))
			// Don't fail the entire scan if deletion cleanup fails
			return 0
		}
	}

	return objectsDeleted
}

// performS3ObjectScan scans and processes all S3 objects and folders.
func (s *Service) performS3ObjectScan(
	ctx context.Context, bucketName string, bucketID, scanJobID int32, objectCount, objectsCreated, objectsUpdated *int,
) error {
	// Use ListObjectsV2 to get all objects
	paginator := s3.NewListObjectsV2Paginator(s.s3Client, &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(s.cfg.S3.Prefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to list objects: %w", err)
		}

		s.processPageObjects(ctx, bucketID, scanJobID, page.Contents, objectCount, objectsCreated, objectsUpdated)
		s.processPageFolders(ctx, bucketID, page.CommonPrefixes, objectsCreated, objectsUpdated)
	}

	return nil
}

// processPageObjects processes a batch of S3 objects from a page.
func (s *Service) processPageObjects(
	ctx context.Context, bucketID, scanJobID int32, objects []types.Object,
	objectCount, objectsCreated, objectsUpdated *int,
) {
	for _, obj := range objects {
		isNew, err := s.processObject(ctx, bucketID, obj)
		if err != nil {
			s.log.Error("Failed to process object",
				slog.String("key", aws.ToString(obj.Key)),
				slog.String("error", err.Error()))
			continue
		}

		// Track creation vs update statistics
		if isNew {
			*objectsCreated++
		} else {
			*objectsUpdated++
		}
		*objectCount++

		// Update progress every 100 objects
		if *objectCount%100 == 0 {
			_, err := s.queries.UpdateScanJobProgress(ctx, database.UpdateScanJobProgressParams{
				ID:             scanJobID,
				ObjectsScanned: sql.NullInt32{Int32: int32(min(*objectCount, math.MaxInt32)), Valid: true},
			})
			if err != nil {
				s.log.Error("Failed to update scan job progress", slog.String("error", err.Error()))
			}
		}
	}
}

// processPageFolders processes a batch of S3 folder prefixes from a page.
func (s *Service) processPageFolders(
	ctx context.Context, bucketID int32, prefixes []types.CommonPrefix, objectsCreated, objectsUpdated *int,
) {
	for _, prefix := range prefixes {
		isNew, err := s.processFolder(ctx, bucketID, aws.ToString(prefix.Prefix))
		if err != nil {
			s.log.Error("Failed to process folder",
				slog.String("prefix", aws.ToString(prefix.Prefix)),
				slog.String("error", err.Error()))
			continue
		}

		// Track folder creation vs update statistics
		if isNew {
			*objectsCreated++
		} else {
			*objectsUpdated++
		}
	}
}

// performPhase1BucketMarking marks all existing buckets for deletion validation.
func (s *Service) performPhase1BucketMarking(ctx context.Context) error {
	s.log.Debug("Phase 1: Marking all buckets for validation")
	if err := s.queries.MarkAllBucketsForDeletion(ctx); err != nil {
		s.log.Error("Failed to mark all buckets for deletion", slog.String("error", err.Error()))
		return fmt.Errorf("failed to mark buckets for validation: %w", err)
	}
	return nil
}

// performPhase2BucketValidation validates discovered buckets and unmarks accessible ones.
func (s *Service) performPhase2BucketValidation(ctx context.Context, discoveredBuckets []string) (int, int, int) {
	s.log.Debug("Phase 2: Validating discovered buckets")
	
	bucketsValidated := 0
	bucketsMarkedInaccessible := 0
	bucketValidationErrors := 0

	for _, bucketName := range discoveredBuckets {
		bucketsValidated++

		// Get bucket record
		bucket, err := s.queries.GetBucket(ctx, bucketName)
		if err != nil {
			s.log.Debug("Bucket not found in database during validation",
				slog.String("bucket", bucketName))
			continue
		}

		// Test accessibility with retries (unless validation is skipped)
		var accessErr error
		if !s.cfg.S3.SkipBucketValidation {
			for retry := range s.cfg.BucketSync.MaxRetries {
				accessErr = s.validateBucketAccessibility(ctx, bucketName)
				if accessErr == nil {
					break
				}

				if retry < s.cfg.BucketSync.MaxRetries-1 {
					s.log.Debug("Retrying bucket accessibility check",
						slog.String("bucket", bucketName),
						slog.Int("retry", retry+1))
					time.Sleep(time.Second * time.Duration(retry+1)) // Exponential backoff
				}
			}
		} else {
			s.log.Debug("Skipping bucket validation", slog.String("bucket", bucketName))
		}

		if accessErr != nil {
			// Bucket is not accessible - record error and mark as inaccessible
			bucketsMarkedInaccessible++
			bucketValidationErrors++

			if err := s.queries.UpdateBucketAccessError(ctx, database.UpdateBucketAccessErrorParams{
				ID:          bucket.ID,
				AccessError: sql.NullString{String: accessErr.Error(), Valid: true},
			}); err != nil {
				s.log.Error("Failed to update bucket access error",
					slog.String("bucket", bucketName),
					slog.String("error", err.Error()))
			}
		} else {
			// Bucket is accessible - unmark for deletion
			if err := s.queries.UnmarkBucketForDeletion(ctx, bucket.ID); err != nil {
				s.log.Error("Failed to unmark bucket for deletion",
					slog.String("bucket", bucketName),
					slog.String("error", err.Error()))
			}
		}
	}

	return bucketsValidated, bucketsMarkedInaccessible, bucketValidationErrors
}

// performPhase3BucketCleanup cleans up buckets that have been inaccessible for too long.
func (s *Service) performPhase3BucketCleanup(ctx context.Context) int {
	s.log.Debug("Phase 3: Cleaning up long-term inaccessible buckets")

	// Parse deletion threshold
	deleteThreshold, err := time.ParseDuration(s.cfg.BucketSync.DeleteThreshold)
	if err != nil {
		s.log.Error("Invalid bucket delete threshold",
			slog.String("threshold", s.cfg.BucketSync.DeleteThreshold),
			slog.String("error", err.Error()))
		return 0
	}

	// Get buckets that should be deleted
	bucketsToDelete, err := s.queries.GetBucketsToDelete(ctx, int32(deleteThreshold.Hours()))
	if err != nil {
		s.log.Error("Failed to get buckets to delete", slog.String("error", err.Error()))
		return 0
	}

	// Delete the buckets
	if len(bucketsToDelete) > 0 {
		s.log.Info("Deleting long-term inaccessible buckets",
			slog.Int("count", len(bucketsToDelete)))

		if err := s.queries.DeleteMarkedBuckets(ctx, int32(deleteThreshold.Hours())); err != nil {
			s.log.Error("Failed to delete marked buckets", slog.String("error", err.Error()))
			return 0
		}
		return len(bucketsToDelete)
	}

	return 0
}
