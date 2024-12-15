package app

import (
	"io"
	"log/slog"
	"net/http"

	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sgaunet/s3xplorer/pkg/views"
)

// IndexBucket handles the index request
func (s *App) IndexBucket(w http.ResponseWriter, r *http.Request) {
	var err error
	var f string
	// vars := mux.Vars(request)
	// bucket := vars["bucket"]

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

	lstFolders, err := s.s3svc.GetFolders(f)
	if err != nil {
		slog.Error("IndexBucket: error when called GetFolders", slog.String("error", err.Error()))
		views.RenderError(err.Error()).Render(r.Context(), w)
		return
	}
	objects, err := s.s3svc.GetObjects(f)
	if err != nil {
		slog.Error("IndexBucket: error when called GetObjects", slog.String("error", err.Error()))
		views.RenderError(err.Error()).Render(r.Context(), w)
		return
	}

	views.RenderIndex(lstFolders, objects, f).Render(r.Context(), w)
}

func (s *App) DownloadFile(w http.ResponseWriter, r *http.Request) {
	var err error
	// vars := mux.Vars(request)
	// bucket := vars["bucket"]
	// key := vars["key"]

	keys, ok := r.URL.Query()["key"]
	if !ok || len(keys[0]) < 1 {
		s.log.Error("Url Param 'key' is missing")
		views.RenderError("Url Param 'key' is missing").Render(r.Context(), w)
		return
	}

	// Query()["key"] will return an array of items,
	// we only want the single item.
	key := keys[0]

	if s.cfg.Prefix != "" {
		if !strings.HasPrefix(key, s.cfg.Prefix) {
			s.log.Error("DownloadFile: Invalid key")
			views.RenderError("Invalid key").Render(r.Context(), w)
			return
		}
	}

	p := s3.GetObjectInput{
		Bucket: &s.cfg.Bucket,
		Key:    &key,
	}
	o, err := s.awsS3Client.GetObject(r.Context(), &p)
	if err != nil {
		s.log.Error("DownloadFile: error when called GetObject", slog.String("error", err.Error()))
		views.RenderError(err.Error()).Render(r.Context(), w)
		return
	}
	defer o.Body.Close()

	w.Header().Set("Content-Disposition", "attachment; filename="+key)
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	_, err = io.Copy(w, o.Body)
	if err != nil {
		s.log.Error("DownloadFile: error when called Copy", slog.String("error", err.Error()))
		views.RenderError(err.Error()).Render(r.Context(), w)
		return
	}
}

// RestoreHandler restores an object from Glacier
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
			views.RenderError("Invalid key").Render(r.Context(), w)
			return
		}
	}

	err = s.s3svc.RestoreObject(key)
	if err != nil {
		s.log.Error("RestoreHandler: error when called RestoreObject", slog.String("error", err.Error()))
		views.RenderError(err.Error()).Render(r.Context(), w)
		return
	}
	http.Redirect(w, r, "/?folder="+f, http.StatusTemporaryRedirect)
}
