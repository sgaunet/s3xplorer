package views

import (
	"embed"
	"net/http"
)

//go:embed static
var staticCSS embed.FS

//go:embed static/file-heart.png
var faviconFS []byte

type Views struct {
	staticHandler http.Handler
}

func NewViews() *Views {
	var staticFS = http.FS(staticCSS)
	fsStatic := http.FileServer(staticFS)
	return &Views{
		staticHandler: fsStatic,
	}
}
