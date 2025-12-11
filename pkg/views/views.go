package views

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:generate go tool github.com/a-h/templ/cmd/templ generate

//go:embed static
var staticCSS embed.FS

//go:embed static/file-heart.png
var faviconFS []byte

// StaticHandler serves static files for the web interface.
var StaticHandler http.Handler

func init() {
	// Create a sub-filesystem rooted at "static/" directory
	staticSubFS, err := fs.Sub(staticCSS, "static")
	if err != nil {
		panic(err)
	}
	// Strip "/static" prefix and serve from the sub-filesystem
	StaticHandler = http.StripPrefix("/static", http.FileServer(http.FS(staticSubFS)))
}
