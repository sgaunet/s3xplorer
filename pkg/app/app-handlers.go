package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sgaunet/s3xplorer/pkg/dto"
	"github.com/sgaunet/s3xplorer/pkg/views"
)

// Package-level error definitions.
var (
	// ErrMissingKeyParam is returned when the required key URL parameter is missing.
	ErrMissingKeyParam = errors.New("URL parameter 'key' is missing")
	// ErrBucketNotAccessible is returned when the requested bucket is not accessible.
	ErrBucketNotAccessible = errors.New("bucket is not accessible")

	// ErrInvalidKey is returned when a key does not match the required prefix.
	ErrInvalidKey = errors.New("invalid key prefix")

	// ErrBucketLocked is returned when bucket changes are not permitted.
	ErrBucketLocked = errors.New("bucket changes are not permitted when a bucket is explicitly defined in configuration")
)

// IndexBucket handles the index request.
// handleBucketSwitch processes bucket switching requests and redirects appropriately.
func (s *App) handleBucketSwitch(ctx context.Context, w http.ResponseWriter, r *http.Request) (bool, error) {
	switchBucket, hasSwitchParam := r.URL.Query()["switchBucket"]
	if !hasSwitchParam || len(switchBucket[0]) < 1 {
		return false, nil // No bucket switch requested
	}

	// Check if bucket switching is allowed
	if s.cfg.S3.BucketLocked {
		// If bucket is locked (specified in config), don't allow changes
		s.log.Warn("Attempted to switch buckets when bucket is locked in config",
			slog.String("current", s.cfg.S3.Bucket),
			slog.String("requested", switchBucket[0]))

		// Render an error page explaining that bucket is locked
		return true, ErrBucketLocked
	}

	// Bucket switching is allowed, proceed with the change
	newBucket := switchBucket[0]
	s.log.Info("Switching bucket", slog.String("to", newBucket))

	// Check if the requested bucket is accessible by getting it from the accessible buckets list
	accessibleBuckets, err := s.dbsvc.GetBuckets(ctx)
	if err != nil {
		s.log.Error("Failed to get accessible buckets", slog.String("error", err.Error()))
		return true, fmt.Errorf("failed to verify bucket accessibility: %w", err)
	}

	// Check if the requested bucket is in the accessible buckets list
	bucketAccessible := false
	for _, bucket := range accessibleBuckets {
		if bucket.Name == newBucket {
			bucketAccessible = true
			break
		}
	}

	if !bucketAccessible {
		s.log.Warn("Attempted to access inaccessible bucket",
			slog.String("bucket", newBucket))
		return true, fmt.Errorf("%w: %s", ErrBucketNotAccessible, newBucket)
	}

	// Update the bucket in the s3svc service
	s.s3svc.SwitchBucket(newBucket)

	// Also update the bucket in the App's config to ensure consistency
	s.cfg.S3.Bucket = newBucket
	s.cfg.S3.Prefix = ""

	// Redirect to the root of the new bucket
	http.Redirect(w, r, "/", http.StatusSeeOther)
	return true, nil // Handled bucket switch
}

// checkEmptyBucket checks if the bucket is empty or needs redirection.
func (s *App) checkEmptyBucket(ctx context.Context, w http.ResponseWriter, r *http.Request) bool {
	// Check if the bucket is empty, if so redirect to bucket selection
	if s.cfg.S3.Bucket == "" {
		s.log.Info("No bucket configured, redirecting to bucket selection")
		http.Redirect(w, r, "/buckets", http.StatusSeeOther)
		return true // Handled with redirect
	}

	// Only check if bucket is empty when not using a prefix filter
	if s.cfg.S3.Prefix == "" {
		count, err := s.dbsvc.CountObjects(ctx, s.cfg.S3.Bucket, s.cfg.S3.Prefix)
		if err != nil {
			s.log.Error("Error checking if bucket is empty", slog.String("error", err.Error()))
			// Continue anyway, we'll show errors on the main page
			return false
		}

		if count == 0 {
			s.log.Info("Bucket is empty, redirecting to bucket selection",
				slog.String("bucket", s.cfg.S3.Bucket))
			http.Redirect(w, r, "/buckets", http.StatusSeeOther)
			return true // Handled with redirect
		}
	}

	return false // No redirect needed
}

// getAndValidateFolder extracts and validates the folder parameter from the request.
func (s *App) getAndValidateFolder(r *http.Request) string {
	// Start with the configured prefix as default
	folderPath := s.cfg.S3.Prefix

	// Check if a folder parameter was provided
	folder, ok := r.URL.Query()["folder"]
	if ok && len(folder[0]) > 0 {
		folderPath = folder[0]

		// Ensure folder respects prefix restrictions if a prefix is set
		if s.cfg.S3.Prefix != "" && !strings.HasPrefix(folderPath, s.cfg.S3.Prefix) {
			folderPath = s.cfg.S3.Prefix // Reset to prefix if validation fails
		}
	}

	s.log.Debug("Using folder path", slog.String("path", folderPath))
	return folderPath
}

// loadAndRenderBucketContents fetches and renders the bucket contents using hierarchical navigation with pagination.
func (s *App) loadAndRenderBucketContents(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	folderPath string,
) error {
	// Parse pagination parameters
	page, err := ParsePaginationParams(r)
	if err != nil {
		// Invalid page parameter, redirect to page 1
		s.log.Warn("Invalid page parameter", slog.String("error", err.Error()))
		redirectURL := fmt.Sprintf("/?folder=%s&page=1", folderPath)
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return nil
	}

	// Get paginated direct children (immediate subfolders and files)
	const pageSize = 50
	folders, files, totalFolders, totalFiles, err := s.dbsvc.GetDirectChildrenPaginated(
		ctx, s.cfg.S3.Bucket, folderPath, page, pageSize,
	)
	if err != nil {
		s.log.Error("Error getting paginated children", slog.String("error", err.Error()))
		return fmt.Errorf("failed to get paginated children: %w", err)
	}

	// Calculate pagination metadata
	totalItems := totalFolders + totalFiles
	paging := dto.NewPaginationInfo(totalItems, pageSize, page)

	// Validate page number against actual total pages
	validPage := ValidatePageNumber(page, paging.TotalPages)
	if page != validPage {
		// Page is out of bounds, redirect to page 1
		s.log.Debug("Page out of bounds, redirecting",
			slog.Int("requested", page),
			slog.Int("valid", validPage),
			slog.Int("totalPages", paging.TotalPages))
		redirectURL := fmt.Sprintf("/?folder=%s&page=1", folderPath)
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		return nil
	}

	// Build breadcrumb navigation
	breadcrumbs := s.dbsvc.BuildBreadcrumbs(folderPath)

	// Render the index page with hierarchical navigation and pagination
	err = views.RenderIndexHierarchical(
		folders, files, folderPath, breadcrumbs, s.cfg, &paging,
	).Render(ctx, w)
	if err != nil {
		s.log.Error("Failed to render index page", slog.String("error", err.Error()))
		return fmt.Errorf("error rendering index page: %w", err)
	}

	return nil
}

// IndexBucket handles the index request with reduced complexity.
func (s *App) IndexBucket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Check if database is available
	if s.dbsvc == nil || !s.IsDatabaseHealthy() {
		s.renderDatabaseUnavailablePage(ctx, w)
		return
	}

	// Check if we're trying to switch buckets
	handled, err := s.handleBucketSwitch(ctx, w, r)
	if err != nil {
		s.renderErrorPage(ctx, w, err.Error())
		return
	}
	if handled {
		return // Request was handled by bucket switch logic
	}

	// Check if we need to redirect for empty bucket
	redirected := s.checkEmptyBucket(ctx, w, r)
	if redirected {
		return // Request was redirected
	}

	// Get and validate folder path
	folderPath := s.getAndValidateFolder(r)

	// Load and render bucket contents with pagination
	err = s.loadAndRenderBucketContents(ctx, w, r, folderPath)
	if err != nil {
		s.renderErrorPage(ctx, w, err.Error())
	}
}

// renderErrorPage is a helper function to render an error page and handle any rendering errors.
func (s *App) renderErrorPage(ctx context.Context, w http.ResponseWriter, message string) {
	if err := views.RenderError(message).Render(ctx, w); err != nil {
		s.log.Error("Failed to render error page", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// extractAndValidateKey extracts the key parameter from the request and validates it.
func (s *App) extractAndValidateKey(r *http.Request) (string, error) {
	keys, ok := r.URL.Query()["key"]
	if !ok || len(keys[0]) < 1 {
		return "", ErrMissingKeyParam
	}

	// Query()["key"] will return an array of items, we only want the single item.
	key := keys[0]

	// Validate the key has the correct prefix if configured
	if s.cfg.S3.Prefix != "" && !strings.HasPrefix(key, s.cfg.S3.Prefix) {
		return "", fmt.Errorf("%w: does not have required prefix '%s'", ErrInvalidKey, s.cfg.S3.Prefix)
	}

	return key, nil
}

// downloadS3Object downloads an object from S3 and streams it to the HTTP response.
func (s *App) downloadS3Object(ctx context.Context, w http.ResponseWriter, key string) error {
	p := s3.GetObjectInput{
		Bucket: &s.cfg.S3.Bucket,
		Key:    &key,
	}

	o, err := s.awsS3Client.GetObject(ctx, &p)
	if err != nil {
		return fmt.Errorf("error getting object from S3: %w", err)
	}
	defer o.Body.Close() //nolint:errcheck

	w.Header().Set("Content-Disposition", "attachment; filename="+key)
	// Handle ContentType which is a pointer
	contentType := "application/octet-stream" // Default content type
	if o.ContentType != nil {
		contentType = *o.ContentType
	}
	w.Header().Set("Content-Type", contentType)

	_, err = io.Copy(w, o.Body)
	if err != nil {
		return fmt.Errorf("error copying S3 object to response: %w", err)
	}

	return nil
}

// DownloadFile handles the download request for a specific file from S3.
func (s *App) DownloadFile(w http.ResponseWriter, r *http.Request) {
	// Extract and validate the key parameter
	key, err := s.extractAndValidateKey(r)
	if err != nil {
		s.log.Error("DownloadFile: key validation failed", slog.String("error", err.Error()))
		s.renderErrorPage(r.Context(), w, err.Error())
		return
	}

	// Download the object from S3
	err = s.downloadS3Object(r.Context(), w, key)
	if err != nil {
		s.log.Error("DownloadFile: download failed", slog.String("error", err.Error()))
		s.renderErrorPage(r.Context(), w, err.Error())
		return
	}
}

// RestoreHandler restores an object from Glacier.
func (s *App) RestoreHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var f string
	keys, ok := r.URL.Query()["key"]

	if !ok || len(keys[0]) < 1 {
		return
	}
	folder, ok := r.URL.Query()["folder"]
	if !ok || len(folder[0]) < 1 {
		f = ""
	} else {
		// Query()["key"] will return an array of items,
		// we only want the single item.
		f = folder[0]
	}
	// Query()["key"] will return an array of items,
	// we only want the single item.
	key := keys[0]
	s.log.Debug("RestoreHandler", slog.String("key", key), slog.String("f", f))

	if s.cfg.S3.Prefix != "" {
		if !strings.HasPrefix(key, s.cfg.S3.Prefix) {
			s.log.Error("RestoreHandler: Invalid key")
			if renderErr := views.RenderError("Invalid key").Render(r.Context(), w); renderErr != nil {
				s.log.Error("Failed to render error page", slog.String("error", renderErr.Error()))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
			return
		}
	}

	err = s.s3svc.RestoreObject(r.Context(), key)
	if err != nil {
		s.log.Error("RestoreHandler: error when called RestoreObject", slog.String("error", err.Error()))
		if renderErr := views.RenderError(err.Error()).Render(r.Context(), w); renderErr != nil {
			s.log.Error("Failed to render error page", slog.String("error", renderErr.Error()))
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}
	http.Redirect(w, r, "/?folder="+f, http.StatusTemporaryRedirect)
}

// HealthCheckHandler provides overall application health status.
func (s *App) HealthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	health := make(map[string]any)

	// Check database health
	if s.dbHealth != nil {
		dbHealth := s.dbHealth.GetHealthInfo()
		health["database"] = dbHealth
		health["overall"] = dbHealth.Status
	} else {
		health["database"] = map[string]any{
			"status":      "unhealthy",
			"last_error":  "Database not configured",
			"is_connected": false,
		}
		health["overall"] = "unhealthy"
	}

	// Set appropriate HTTP status
	statusCode := http.StatusOK
	if health["overall"] != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(health); err != nil {
		s.log.Error("Failed to encode health response", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// DatabaseHealthHandler provides detailed database health information.
func (s *App) DatabaseHealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if s.dbHealth == nil {
		// Render database unavailable page
		s.renderDatabaseUnavailablePage(ctx, w)
		return
	}

	healthInfo := s.dbHealth.GetHealthInfo()

	// For HTML requests, render a user-friendly page
	if strings.Contains(r.Header.Get("Accept"), "text/html") {
		if healthInfo.IsConnected {
			if err := views.RenderDatabaseHealthy(healthInfo.LastCheck).Render(ctx, w); err != nil {
				s.log.Error("Failed to render database health page", slog.String("error", err.Error()))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		} else {
			s.renderDatabaseUnavailablePage(ctx, w)
		}
		return
	}

	// For JSON requests, return structured data
	w.Header().Set("Content-Type", "application/json")
	statusCode := http.StatusOK
	if !healthInfo.IsConnected {
		statusCode = http.StatusServiceUnavailable
	}
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(healthInfo); err != nil {
		s.log.Error("Failed to encode database health response", slog.String("error", err.Error()))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}

// renderDatabaseUnavailablePage renders an error page when database is unavailable.
func (s *App) renderDatabaseUnavailablePage(ctx context.Context, w http.ResponseWriter) {
	w.WriteHeader(http.StatusServiceUnavailable)
	if err := views.RenderDatabaseUnavailable().Render(ctx, w); err != nil {
		s.log.Error("Failed to render database unavailable page", slog.String("error", err.Error()))
		http.Error(w, "Database is currently unavailable. Please try again later.", http.StatusServiceUnavailable)
	}
}
