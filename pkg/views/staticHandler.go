package views

import "net/http"

// GetStaticHandler returns the static handler (for CSS and images)
func (v *Views) GetStaticHandler() http.Handler {
	return v.staticHandler
}
