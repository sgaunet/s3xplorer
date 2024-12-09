package app

import (
	"net/http"

	"github.com/sgaunet/s3xplorer/pkg/views"
)

func (s *App) SearchHandler(response http.ResponseWriter, request *http.Request) {
	var err error
	var searchFile string

	searchstr, ok := request.URL.Query()["searchstr"]
	if !ok || len(searchstr[0]) < 1 {
		searchFile = ""
	} else {
		searchFile = searchstr[0]
	}

	objects, err := s.s3svc.SearchObjects(s.cfg.Prefix, searchFile)
	if err != nil {
		s.views.HandlerError(response, err.Error())
		return
	}

	err = s.views.RenderSearch(response, views.SearchData{
		ActualFolder: s.cfg.Prefix,
		Objects:      objects,
		SearchStr:    searchFile,
	})
	if err != nil {
		s.views.HandlerError(response, err.Error())
		return
	}
}
