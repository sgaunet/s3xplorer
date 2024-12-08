package views

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"text/template"
)

// HandlerError renders the error page
func (v *Views) HandlerError(response http.ResponseWriter, errorStr string) {
	tmplContent, err := fs.ReadFile(templatesFS, "templates/error.html")
	if err != nil {
		panic(err)
	}
	// Create a new template, add functions, and parse the template content
	t, err := template.New("error.html").Funcs(template.FuncMap{
		"basename": func(path string) string {
			return filepath.Base(path)
		},
	}).Parse(string(tmplContent))
	if err != nil {
		panic(err)
	}

	data := dataErr{
		ErrorMsg: errorStr,
	}
	err = t.Execute(response, data)
	if err != nil {
		panic(err)
	}
}
