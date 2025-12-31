package app

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
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

	// 3. Parse form data
	err := r.ParseForm()
	if err != nil {
		s.log.Error("Failed to parse form", slog.String("error", err.Error()))
		s.renderErrorPage(ctx, w, "Failed to parse request")
		return
	}

	// 4. Get folder parameter
	folder := r.FormValue("folder")
	if folder == "" {
		folder = s.cfg.S3.Prefix
	}

	// Validate folder respects prefix restrictions
	if s.cfg.S3.Prefix != "" && !strings.HasPrefix(folder, s.cfg.S3.Prefix) {
		folder = s.cfg.S3.Prefix
	}

	// 5. Get keys to delete (multiple keys from checkboxes)
	keys := r.Form["keys"]
	if len(keys) == 0 {
		s.renderErrorPage(ctx, w, "No files selected for deletion")
		return
	}

	s.log.Info("Delete request",
		slog.String("folder", folder),
		slog.Int("count", len(keys)))

	// 6. Validate all keys respect prefix restrictions
	for _, key := range keys {
		if s.cfg.S3.Prefix != "" && !strings.HasPrefix(key, s.cfg.S3.Prefix) {
			s.log.Warn("Delete attempt outside configured prefix",
				slog.String("key", key),
				slog.String("prefix", s.cfg.S3.Prefix))
			s.renderErrorPage(ctx, w, "Cannot delete files outside configured prefix")
			return
		}
	}

	// 7. Delete from S3
	var deleteErr error
	if len(keys) == 1 {
		// Single delete
		deleteErr = s.s3svc.DeleteObject(ctx, keys[0])
	} else {
		// Bulk delete (more efficient for multiple files)
		deleteErr = s.s3svc.DeleteObjects(ctx, keys)
	}

	if deleteErr != nil {
		s.log.Error("Failed to delete from S3", slog.String("error", deleteErr.Error()))
		s.renderErrorPage(ctx, w, "Delete failed: "+deleteErr.Error())
		return
	}

	// 8. Sync to database (immediate sync for UX)
	var syncErr error
	if len(keys) == 1 {
		syncErr = s.dbsvc.SyncDeletedObject(ctx, s.cfg.S3.Bucket, keys[0])
	} else {
		syncErr = s.dbsvc.SyncDeletedObjects(ctx, s.cfg.S3.Bucket, keys)
	}

	if syncErr != nil {
		// Log error but don't fail the delete (background scanner will sync eventually)
		s.log.Error("Failed to sync delete to database", slog.String("error", syncErr.Error()))
	}

	// 9. Redirect back to folder
	redirectURL := fmt.Sprintf("/?folder=%s&page=1", url.QueryEscape(folder))
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}
