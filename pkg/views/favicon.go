// Package views provides HTML templates and view rendering functions for the application
package views

import "net/http"

// FaviconHandler handles the favicon.ico request
func FaviconHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=7776000")
	w.Write(faviconFS) //nolint:errcheck
}
