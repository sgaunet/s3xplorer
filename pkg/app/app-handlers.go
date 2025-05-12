package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sgaunet/s3xplorer/pkg/views"
)

// Package-level error definitions.
var (
	// ErrMissingKeyParam is returned when the required key URL parameter is missing.
	ErrMissingKeyParam = errors.New("URL parameter 'key' is missing")

	// ErrInvalidKey is returned when a key does not match the required prefix.
	ErrInvalidKey = errors.New("invalid key prefix")
)

// IndexBucket handles the index request.
func (s *App) IndexBucket(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var err error
	var f string

	// Check if we're trying to switch buckets
	switchBucket, hasSwitchParam := r.URL.Query()["switchBucket"]
	if hasSwitchParam && len(switchBucket[0]) > 0 {
		// Check if bucket switching is allowed
		if s.cfg.BucketLocked {
			// If bucket is locked (specified in config), don't allow changes
			s.log.Warn("Attempted to switch buckets when bucket is locked in config", 
				slog.String("current", s.cfg.Bucket),
				slog.String("requested", switchBucket[0]))
			
			// Render an error page explaining that bucket is locked
			s.renderErrorPage(ctx, w, "Bucket changes are not permitted when a bucket is explicitly defined in configuration")
			return
		}
		
		// Bucket switching is allowed, proceed with the change
		newBucket := switchBucket[0]
		s.log.Info("Switching bucket", slog.String("to", newBucket))
		
		// Update the bucket in the s3svc service
		s.s3svc.SwitchBucket(newBucket)
		
		// Also update the bucket in the App's config to ensure consistency
		// This is crucial for download functionality which uses App's config directly
		s.cfg.Bucket = newBucket
		s.cfg.Prefix = ""
		
		// Redirect to the root of the new bucket
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	// Check if the bucket is empty, if so redirect to the bucket selection page
	if s.cfg.Bucket == "" {
		s.log.Info("No bucket configured, redirecting to bucket selection")
		http.Redirect(w, r, "/buckets", http.StatusSeeOther)
		return
	}

	// Check if the bucket exists but is empty (and not just because of a filter)
	if s.cfg.Prefix == "" { 
		isEmpty, err := s.s3svc.IsBucketEmpty(ctx)
		if err != nil {
			s.log.Error("Error checking if bucket is empty", slog.String("error", err.Error()))
			// Continue anyway, we'll show an error on the main page if there are issues
		} else if isEmpty {
			s.log.Info("Bucket is empty, redirecting to bucket selection", slog.String("bucket", s.cfg.Bucket))
			http.Redirect(w, r, "/buckets", http.StatusSeeOther)
			return
		}
	}

	// Handle folder parameter
	folder, ok := r.URL.Query()["folder"]
	if !ok || len(folder[0]) < 1 {
		f = s.cfg.Prefix
	} else {
		// Query()["key"] will return an array of items,
		// we only want the single item.
		f = folder[0]
		// Check that the folder begins with s.cfg.Prefix if s.cfg.Prefix is not empty
		if s.cfg.Prefix != "" {
			if !strings.HasPrefix(f, s.cfg.Prefix) {
				f = s.cfg.Prefix
			}
		}
	}
	s.log.Debug("IndexBucket", slog.String("f", f))

	lstFolders, err := s.s3svc.GetFolders(ctx, f)
	if err != nil {
		s.log.Error("IndexBucket: error when called GetFolders", slog.String("error", err.Error()))
		s.renderErrorPage(ctx, w, err.Error())
		return
	}
	objects, err := s.s3svc.GetObjects(ctx, f)
	if err != nil {
		s.log.Error("IndexBucket: error when called GetObjects", slog.String("error", err.Error()))
		s.renderErrorPage(ctx, w, err.Error())
		return
	}

	if err := views.RenderIndex(lstFolders, objects, f, s.cfg).Render(ctx, w); err != nil {
		s.log.Error("Failed to render index page", slog.String("error", err.Error()))
		http.Error(w, "Internal server error rendering index page", http.StatusInternalServerError)
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
	if s.cfg.Prefix != "" && !strings.HasPrefix(key, s.cfg.Prefix) {
		return "", fmt.Errorf("%w: does not have required prefix '%s'", ErrInvalidKey, s.cfg.Prefix)
	}

	return key, nil
}

// downloadS3Object downloads an object from S3 and streams it to the HTTP response.
func (s *App) downloadS3Object(ctx context.Context, w http.ResponseWriter, key string) error {
	p := s3.GetObjectInput{
		Bucket: &s.cfg.Bucket,
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

	if s.cfg.Prefix != "" {
		if !strings.HasPrefix(key, s.cfg.Prefix) {
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
