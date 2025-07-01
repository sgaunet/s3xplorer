// Package app implements the web application's core functionality and HTTP request handlers.
package app

import (
	"log/slog"
	"net/http"

	"github.com/sgaunet/s3xplorer/pkg/views"
)

// SearchHandler handles the search request.
func (s *App) SearchHandler(w http.ResponseWriter, r *http.Request) {
	var err error
	var searchFile string

	searchstr, ok := r.URL.Query()["searchstr"]
	if !ok || len(searchstr[0]) < 1 {
		searchFile = ""
	} else {
		searchFile = searchstr[0]
	}
	s.log.Debug("SearchHandler", slog.String("searchFile", searchFile))

	// Use PostgreSQL database service for search instead of direct S3 calls
	objects, err := s.dbsvc.SearchObjects(r.Context(), s.cfg.Bucket, searchFile, 1000, 0)
	if err != nil {
		s.log.Error("SearchHandler: error when called SearchObjects", slog.String("error", err.Error()))
		if err := views.RenderError(err.Error()).Render(r.Context(), w); err != nil {
			s.log.Error("Failed to render error page", slog.String("error", err.Error()))
			http.Error(w, "Internal server error rendering error page", http.StatusInternalServerError)
		}
		return
	}

	if err := views.RenderSearch(searchFile, s.cfg.Prefix, objects, s.cfg).Render(r.Context(), w); err != nil {
		s.log.Error("Failed to render search results", slog.String("error", err.Error()))
		http.Error(w, "Internal server error rendering search results", http.StatusInternalServerError)
	}
}
