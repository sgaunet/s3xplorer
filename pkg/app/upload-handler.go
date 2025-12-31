package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

const (
	// MaxUploadSize is the maximum file size allowed (100 MB).
	MaxUploadSize = 100 * 1024 * 1024 // 100 MB
)

var (
	// ErrParseUploadRequest indicates failure to parse the upload form.
	ErrParseUploadRequest = errors.New("failed to parse upload request")
	// ErrNoFileUploaded indicates no file was provided in the upload request.
	ErrNoFileUploaded = errors.New("no file uploaded")
	// ErrFileTooLarge indicates the uploaded file exceeds the size limit.
	ErrFileTooLarge = errors.New("file too large")
	// ErrUploadOutsidePrefix indicates an attempt to upload outside the configured prefix.
	ErrUploadOutsidePrefix = errors.New("cannot upload outside configured prefix")
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

	// 3. Parse and process upload
	if err := s.processUpload(ctx, w, r); err != nil {
		s.renderErrorPage(ctx, w, err.Error())
	}
}

// processUpload handles the actual upload processing logic.
func (s *App) processUpload(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	// Parse multipart form
	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
		s.log.Error("Failed to parse multipart form", slog.String("error", err.Error()))
		return ErrParseUploadRequest
	}

	// Get and validate folder
	folder := s.getValidatedFolder(r)

	// Get uploaded file
	file, header, err := r.FormFile("file")
	if err != nil {
		s.log.Error("Failed to get uploaded file", slog.String("error", err.Error()))
		return ErrNoFileUploaded
	}
	defer file.Close() //nolint:errcheck

	// Validate file size
	if header.Size > MaxUploadSize {
		const bytesPerMB = 1024 * 1024
		return fmt.Errorf("%w (max %d MB)", ErrFileTooLarge, MaxUploadSize/bytesPerMB)
	}

	// Construct and validate S3 key
	key := folder + header.Filename
	if !s.validateKeyPrefix(key) {
		s.log.Warn("Upload attempt outside configured prefix",
			slog.String("key", key),
			slog.String("prefix", s.cfg.S3.Prefix))
		return ErrUploadOutsidePrefix
	}

	// Detect content type
	contentType := s.detectContentType(header)

	s.log.Info("Upload request",
		slog.String("key", key),
		slog.String("contentType", contentType),
		slog.Int64("size", header.Size))

	// Upload to S3
	if err := s.s3svc.UploadObject(ctx, key, file, contentType, header.Size); err != nil {
		s.log.Error("Failed to upload to S3", slog.String("error", err.Error()))
		return fmt.Errorf("upload failed: %w", err)
	}

	// Sync to database (log errors but don't fail)
	if err := s.dbsvc.SyncUploadedObject(ctx, s.cfg.S3.Bucket, key, header.Size, "", "STANDARD"); err != nil {
		s.log.Error("Failed to sync upload to database", slog.String("error", err.Error()))
	}

	// Redirect back to folder
	redirectURL := fmt.Sprintf("/?folder=%s&page=1", url.QueryEscape(folder))
	http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	return nil
}

// getValidatedFolder extracts and validates the folder parameter from form data.
func (s *App) getValidatedFolder(r *http.Request) string {
	folder := r.FormValue("folder")
	if folder == "" {
		folder = s.cfg.S3.Prefix
	}

	// Validate folder respects prefix restrictions
	if s.cfg.S3.Prefix != "" && !strings.HasPrefix(folder, s.cfg.S3.Prefix) {
		s.log.Warn("Upload attempt outside configured prefix",
			slog.String("folder", folder),
			slog.String("prefix", s.cfg.S3.Prefix))
		return s.cfg.S3.Prefix
	}

	return folder
}

// validateKeyPrefix checks if a key respects the configured prefix.
func (s *App) validateKeyPrefix(key string) bool {
	if s.cfg.S3.Prefix == "" {
		return true
	}
	return strings.HasPrefix(key, s.cfg.S3.Prefix)
}

// detectContentType determines the content type from the file header.
func (s *App) detectContentType(header *multipart.FileHeader) string {
	contentType := header.Header.Get("Content-Type")
	if contentType != "" {
		return contentType
	}

	// Fallback to detection based on file extension
	contentType = mime.TypeByExtension(filepath.Ext(header.Filename))
	if contentType != "" {
		return contentType
	}

	return "application/octet-stream"
}
