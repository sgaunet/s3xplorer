package views

import "net/http"

// FaviconHandler handles the favicon.ico request
func FaviconHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=7776000")
	w.Write(faviconFS)
}
