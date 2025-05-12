package views

import (
	"embed"
	"net/http"
)

//go:generate go tool github.com/a-h/templ/cmd/templ generate

//go:embed static
var staticCSS embed.FS

//go:embed static/file-heart.png
var faviconFS []byte

// StaticHandler serves static files for the web interface
var StaticHandler http.Handler

func init() {
	var staticFS = http.FS(staticCSS)
	StaticHandler = http.FileServer(staticFS)
}
