// Package views provides HTML templates and view rendering functions for the application
package views

import (
	"log/slog"
	"net/http"
)

// FaviconHandler handles the favicon.ico request.
func FaviconHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=7776000")
	_, err := w.Write(faviconFS)
	if err != nil {
		// Log error but continue since there's nothing the client can do about a favicon error
		slog.Debug("Failed to serve favicon", slog.String("error", err.Error()))
		// We don't return an HTTP error for favicon failures as they're not critical
	}
}
