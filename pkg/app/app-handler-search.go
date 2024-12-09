package app

import (
	"log/slog"
	"net/http"

	"github.com/sgaunet/s3xplorer/pkg/views"
)

// SearchHandler handles the search request
func (s *App) SearchHandler(response http.ResponseWriter, request *http.Request) {
	var err error
	var searchFile string

	searchstr, ok := request.URL.Query()["searchstr"]
	if !ok || len(searchstr[0]) < 1 {
		searchFile = ""
	} else {
		searchFile = searchstr[0]
	}
	s.log.Debug("SearchHandler", slog.String("searchFile", searchFile))

	objects, err := s.s3svc.SearchObjects(s.cfg.Prefix, searchFile)
	if err != nil {
		slog.Error("SearchHandler: error when called SearchObjects", slog.String("error", err.Error()))
		s.views.HandlerError(response, err.Error())
		return
	}

	err = s.views.RenderSearch(response, views.SearchData{
		ActualFolder: s.cfg.Prefix,
		Objects:      objects,
		SearchStr:    searchFile,
	})
	if err != nil {
		slog.Error("SearchHandler: error when called RenderSearch", slog.String("error", err.Error()))
		s.views.HandlerError(response, err.Error())
		return
	}
}
