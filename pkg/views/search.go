package views

import (
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"text/template"

	"github.com/sgaunet/s3xplorer/pkg/dto"
)

// SearchData is the data structure used to render the search page
type SearchData struct {
	ActualFolder string
	Objects      []dto.S3Object
	SearchStr    string
}

// RenderSearch renders the search page
func (v *Views) RenderSearch(w io.Writer, data SearchData) error {
	tmplContent, err := fs.ReadFile(templatesFS, "templates/search.html")
	if err != nil {
		return fmt.Errorf("error reading file: %s", err)
	}
	menuContent, err := fs.ReadFile(templatesFS, "templates/menu.html")
	if err != nil {
		return fmt.Errorf("error reading file: %s", err)
	}

	// Create a new template, add functions, and parse the template content
	t, err := template.New("search.html").Funcs(template.FuncMap{
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
