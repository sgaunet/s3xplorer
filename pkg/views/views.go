package views

import (
	"embed"
	"net/http"
)

//go:embed static
var staticCSS embed.FS

//go:embed static/file-heart.png
var faviconFS []byte

// staticHandler is a http.Handler that serves static files
var StaticHandler http.Handler

func init() {
	var staticFS = http.FS(staticCSS)
	StaticHandler = http.FileServer(staticFS)
}
