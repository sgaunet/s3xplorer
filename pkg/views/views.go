package views

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path/filepath"

	"github.com/sgaunet/s3xplorer/pkg/dto"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static
var staticCSS embed.FS

//go:embed static/file-heart.png
var faviconFS []byte

type dataErr struct {
	ErrorMsg string
}

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

// IndexData is the data structure used to render the index page
type IndexData struct {
	ActualFolder string
	Folders      []dto.S3Object
	Objects      []dto.S3Object
}

// RenderIndex renders the index page
func (v *Views) RenderIndex(w io.Writer, data IndexData) error {
	tmplContent, err := fs.ReadFile(templatesFS, "templates/bucket-listing.html")
	if err != nil {
		return fmt.Errorf("error reading file: %s", err)
	}
	menuContent, err := fs.ReadFile(templatesFS, "templates/menu.html")
	if err != nil {
		return fmt.Errorf("error reading file: %s", err)
	}

	// Create a new template, add functions, and parse the template content
	t, err := template.New("bucket-listing.html").Funcs(template.FuncMap{
		"basename": func(path string) string {
			return filepath.Base(path)
		},
	}).Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("error parsing template: %s", err)
	}

	// Parse the menu template
	_, err = t.New("menu").Parse(string(menuContent))
	if err != nil {
		return fmt.Errorf("error parsing menu template: %s", err)
	}

	err = t.Execute(w, data)
	if err != nil {
		return fmt.Errorf("error when generating template error: %s", err)
	}
	return nil
}
