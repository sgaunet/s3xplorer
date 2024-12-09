package app

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sgaunet/s3xplorer/pkg/views"
)

// IndexBucket handles the index request
func (s *App) IndexBucket(response http.ResponseWriter, request *http.Request) {
	var err error
	var f string
	// vars := mux.Vars(request)
	// bucket := vars["bucket"]

	folder, ok := request.URL.Query()["folder"]
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
		s.views.HandlerError(response, err.Error())
		return
	}
	objects, err := s.s3svc.GetObjects(f)
	if err != nil {
		slog.Error("IndexBucket: error when called GetObjects", slog.String("error", err.Error()))
		s.views.HandlerError(response, err.Error())
		return
	}

	err = s.views.RenderIndex(response, views.IndexData{
		ActualFolder: f,
		Folders:      lstFolders,
		Objects:      objects,
	})
	if err != nil {
		slog.Error("IndexBucket: error when called RenderIndex", slog.String("error", err.Error()))
		s.views.HandlerError(response, err.Error())
		return
	}
}

func (s *App) DownloadFile(w http.ResponseWriter, request *http.Request) {
	var err error
	// vars := mux.Vars(request)
	// bucket := vars["bucket"]
	// key := vars["key"]

	keys, ok := request.URL.Query()["key"]
	if !ok || len(keys[0]) < 1 {
		s.log.Error("Url Param 'key' is missing")
		s.views.HandlerError(w, "Url Param 'key' is missing")
		return
	}

	// Query()["key"] will return an array of items,
	// we only want the single item.
	key := keys[0]

	if s.cfg.Prefix != "" {
		if !strings.HasPrefix(key, s.cfg.Prefix) {
			s.log.Error("DownloadFile: Invalid key")
			s.views.HandlerError(w, "Invalid key")
			return
		}
	}

	p := s3.GetObjectInput{
		Bucket: &s.cfg.Bucket,
		Key:    &key,
	}
	o, err := s.awsS3Client.GetObject(context.TODO(), &p)
	if err != nil {
		s.log.Error("DownloadFile: error when called GetObject", slog.String("error", err.Error()))
		s.views.HandlerError(w, err.Error())
		return
	}

	// All the file is read in memory, it's not a good idea for big files
	// TODO: improve this
	buffer, err := io.ReadAll(o.Body)
	if err != nil {
		s.log.Error("DownloadFile: error when called ReadAll", slog.String("error", err.Error()))
		s.views.HandlerError(w, err.Error())
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+key)
	w.Header().Set("Content-Type", request.Header.Get("Content-Type"))
	http.ServeContent(w, request, key, time.Now(), bytes.NewReader(buffer))
}

// RestoreHandler restores an object from Glacier
func (s *App) RestoreHandler(w http.ResponseWriter, request *http.Request) {
	var err error
	var f string
	keys, ok := request.URL.Query()["key"]

	if !ok || len(keys[0]) < 1 {
		return
	}
	folder, ok := request.URL.Query()["folder"]
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
			s.views.HandlerError(w, "Invalid key")
			return
		}
	}

	err = s.s3svc.RestoreObject(key)
	if err != nil {
		s.log.Error("RestoreHandler: error when called RestoreObject", slog.String("error", err.Error()))
		s.views.HandlerError(w, err.Error())
		return
	}
	http.Redirect(w, request, "/?folder="+f, http.StatusTemporaryRedirect)
}
