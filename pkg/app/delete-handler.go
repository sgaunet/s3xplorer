package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

var (
	// ErrParseDeleteRequest indicates failure to parse the delete form.
	ErrParseDeleteRequest = errors.New("failed to parse request")
	// ErrNoFilesSelected indicates no files were selected for deletion.
	ErrNoFilesSelected = errors.New("no files selected for deletion")
	// ErrDeleteOutsidePrefix indicates an attempt to delete files outside the configured prefix.
	ErrDeleteOutsidePrefix = errors.New("cannot delete files outside configured prefix")
)

// DeleteHandler handles file deletion requests (single or bulk).
func (s *App) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Check feature flag
	if !s.cfg.S3.EnableDelete {
		s.log.Warn("Delete attempt when feature is disabled")
		s.renderErrorPage(ctx, w, "Delete functionality is disabled")
		return
	}

	// 2. Validate method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 3. Parse and process deletion
	if err := s.processDelete(ctx, w, r); err != nil {
		s.renderErrorPage(ctx, w, err.Error())
	}
}

// processDelete handles the actual deletion processing logic.
func (s *App) processDelete(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// Parse form data
	if err := r.ParseForm(); err != nil {
		s.log.Error("Failed to parse form", slog.String("error", err.Error()))
		return ErrParseDeleteRequest
	}

	// Get and validate folder
	folder := s.getDeleteValidatedFolder(r)

	// Get keys to delete
	keys := r.Form["keys"]
	if len(keys) == 0 {
		return ErrNoFilesSelected
	}

	s.log.Info("Delete request",
		slog.String("folder", folder),
		slog.Int("count", len(keys)))

	// Validate all keys
	if err := s.validateDeleteKeys(keys); err != nil {
		return err
	}

	// Delete from S3
	if err := s.performS3Delete(ctx, keys); err != nil {
		s.log.Error("Failed to delete from S3", slog.String("error", err.Error()))
		return fmt.Errorf("delete failed: %w", err)
	}

	// Sync to database (log errors but don't fail)
	if err := s.performDatabaseDeleteSync(ctx, keys); err != nil {
		s.log.Error("Failed to sync delete to database", slog.String("error", err.Error()))
	}

	// Redirect back to folder
	redirectURL := fmt.Sprintf("/?folder=%s&page=1", url.QueryEscape(folder))
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	return nil
}

// getDeleteValidatedFolder extracts and validates the folder parameter for deletion.
func (s *App) getDeleteValidatedFolder(r *http.Request) string {
	folder := r.FormValue("folder")
	if folder == "" {
		folder = s.cfg.S3.Prefix
	}

	// Validate folder respects prefix restrictions
	if s.cfg.S3.Prefix != "" && !strings.HasPrefix(folder, s.cfg.S3.Prefix) {
		return s.cfg.S3.Prefix
	}

	return folder
}

// validateDeleteKeys validates that all keys respect the configured prefix.
func (s *App) validateDeleteKeys(keys []string) error {
	if s.cfg.S3.Prefix == "" {
		return nil
	}

	for _, key := range keys {
		if !strings.HasPrefix(key, s.cfg.S3.Prefix) {
			s.log.Warn("Delete attempt outside configured prefix",
				slog.String("key", key),
				slog.String("prefix", s.cfg.S3.Prefix))
			return ErrDeleteOutsidePrefix
		}
	}
	return nil
}

// performS3Delete deletes objects from S3 (single or bulk).
func (s *App) performS3Delete(ctx context.Context, keys []string) error {
	if len(keys) == 1 {
		return s.s3svc.DeleteObject(ctx, keys[0])
	}
	return s.s3svc.DeleteObjects(ctx, keys)
}

// performDatabaseDeleteSync syncs deleted objects to the database.
func (s *App) performDatabaseDeleteSync(ctx context.Context, keys []string) error {
	if len(keys) == 1 {
		return s.dbsvc.SyncDeletedObject(ctx, s.cfg.S3.Bucket, keys[0])
	}
	return s.dbsvc.SyncDeletedObjects(ctx, s.cfg.S3.Bucket, keys)
}
