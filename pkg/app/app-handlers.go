package app

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/sgaunet/s3xplorer/pkg/views"
)

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

	lstFolders, err := s.s3svc.GetFolders(f)
	if err != nil {
		s.views.HandlerError(response, err.Error())
		return
	}
	objects, err := s.s3svc.GetObjects(f)
	if err != nil {
		s.views.HandlerError(response, err.Error())
		return
	}

	err = s.views.RenderIndex(response, views.IndexData{
		ActualFolder: f,
		Folders:      lstFolders,
		Objects:      objects,
	})
	if err != nil {
		s.views.HandlerError(response, err.Error())
		return
	}

}

func (s *App) DownloadFile(w http.ResponseWriter, request *http.Request) {
	var err error
	// vars := mux.Vars(request)
	// bucket := vars["bucket"]
	// key := vars["key"]
	var cfg aws.Config

	keys, ok := request.URL.Query()["key"]
	if !ok || len(keys[0]) < 1 {
		s.log.Debugln("Url Param 'key' is missing")
		return
	}

	// Query()["key"] will return an array of items,
	// we only want the single item.
	key := keys[0]

	if s.cfg.Prefix != "" {
		if !strings.HasPrefix(key, s.cfg.Prefix) {
			s.views.HandlerError(w, "Invalid key")
			return
		}
	}

	cfg, err = s.GetAwsConfig()
	if err != nil {
		v := views.NewViews()
		v.HandlerError(w, err.Error())
		return
	}
	// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3
	awsS3Client := s3.NewFromConfig(cfg)

	p := s3.GetObjectInput{
		Bucket: &s.cfg.Bucket,
		Key:    &key,
	}
	o, err := awsS3Client.GetObject(context.TODO(), &p)
	if err != nil {
		v := views.NewViews()
		v.HandlerError(w, err.Error())
		return
	}
	buffer, err := io.ReadAll(o.Body)
	if err != nil {
		v := views.NewViews()
		v.HandlerError(w, err.Error())
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+key)
	w.Header().Set("Content-Type", request.Header.Get("Content-Type"))
	http.ServeContent(w, request, key, time.Now(), bytes.NewReader(buffer))
}

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

	if s.cfg.Prefix != "" {
		if !strings.HasPrefix(key, s.cfg.Prefix) {
			s.views.HandlerError(w, "Invalid key")
			return
		}
	}

	err = s.s3svc.RestoreObject(key)
	if err != nil {
		s.log.Errorln(err.Error())
	}
	s.log.Debugln("f=", f)
	http.Redirect(w, request, "/"+s.cfg.Bucket+"?folder="+f, http.StatusMovedPermanently)
}
