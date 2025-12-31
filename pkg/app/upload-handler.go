package app

import (
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

const (
	// MaxUploadSize is the maximum file size allowed (100 MB).
	MaxUploadSize = 100 * 1024 * 1024 // 100 MB
)

// UploadHandler handles file upload requests.
func (s *App) UploadHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// 1. Check feature flag
	if !s.cfg.S3.EnableUpload {
		s.log.Warn("Upload attempt when feature is disabled")
		s.renderErrorPage(ctx, w, "Upload functionality is disabled")
		return
	}

	// 2. Validate method
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 3. Parse multipart form (limit to MaxUploadSize)
	err := r.ParseMultipartForm(MaxUploadSize)
	if err != nil {
		s.log.Error("Failed to parse multipart form", slog.String("error", err.Error()))
		s.renderErrorPage(ctx, w, "Failed to parse upload request")
		return
	}

	// 4. Get folder parameter from form data (not query params)
	folder := r.FormValue("folder")
	if folder == "" {
		folder = s.cfg.S3.Prefix
	}

	// Validate folder respects prefix restrictions
	if s.cfg.S3.Prefix != "" && !strings.HasPrefix(folder, s.cfg.S3.Prefix) {
		s.log.Warn("Upload attempt outside configured prefix",
			slog.String("folder", folder),
			slog.String("prefix", s.cfg.S3.Prefix))
		folder = s.cfg.S3.Prefix
	}

	// 5. Get uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		s.log.Error("Failed to get uploaded file", slog.String("error", err.Error()))
		s.renderErrorPage(ctx, w, "No file uploaded")
		return
	}
	defer file.Close() //nolint:errcheck

	// 6. Validate file size
	if header.Size > MaxUploadSize {
		const bytesPerMB = 1024 * 1024
		s.renderErrorPage(ctx, w, fmt.Sprintf("File too large (max %d MB)", MaxUploadSize/bytesPerMB))
		return
	}

	// 7. Construct S3 key (folder + filename)
	filename := header.Filename
	key := folder + filename

	// 8. Validate key respects prefix restrictions
	if s.cfg.S3.Prefix != "" && !strings.HasPrefix(key, s.cfg.S3.Prefix) {
		s.log.Warn("Upload attempt outside configured prefix",
			slog.String("key", key),
			slog.String("prefix", s.cfg.S3.Prefix))
		s.renderErrorPage(ctx, w, "Cannot upload outside configured prefix")
		return
	}

	// 9. Detect content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		// Fallback to detection based on file extension
		contentType = mime.TypeByExtension(filepath.Ext(filename))
		if contentType == "" {
			contentType = "application/octet-stream"
		}
	}

	s.log.Info("Upload request",
		slog.String("key", key),
		slog.String("contentType", contentType),
		slog.Int64("size", header.Size))

	// 10. Upload to S3
	err = s.s3svc.UploadObject(ctx, key, file, contentType, header.Size)
	if err != nil {
		s.log.Error("Failed to upload to S3", slog.String("error", err.Error()))
		s.renderErrorPage(ctx, w, "Upload failed: "+err.Error())
		return
	}

	// 11. Sync to database (immediate sync for UX)
	// Note: ETag and StorageClass will be populated by background scanner
	err = s.dbsvc.SyncUploadedObject(ctx, s.cfg.S3.Bucket, key, header.Size, "", "STANDARD")
	if err != nil {
		// Log error but don't fail the upload (background scanner will sync eventually)
		s.log.Error("Failed to sync upload to database", slog.String("error", err.Error()))
	}

	// 12. Redirect back to folder
	redirectURL := fmt.Sprintf("/?folder=%s&page=1", url.QueryEscape(folder))
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
}
