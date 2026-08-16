package views

import (
	"bytes"
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"
)

//go:generate go tool github.com/a-h/templ/cmd/templ generate

//go:embed static
var staticCSS embed.FS

//go:embed static/file-heart.png
var faviconFS []byte

// staticCacheControl is sent for every /static response. Templates append a
// ?v=N cache-buster to their asset URLs, so a long immutable TTL is safe.
const staticCacheControl = "public, max-age=31536000, immutable"

// StaticHandler serves static files for the web interface.
var StaticHandler http.Handler

func init() {
	// Create a sub-filesystem rooted at "static/" directory
	staticSubFS, err := fs.Sub(staticCSS, "static")
	if err != nil {
		panic(err)
	}
	// Strip "/static" prefix and serve from the sub-filesystem
	fileServer := http.StripPrefix("/static", http.FileServer(http.FS(staticSubFS)))
	StaticHandler = &staticHandler{files: staticSubFS, fallback: fileServer}
}

// staticHandler serves embedded assets, preferring a pre-compressed ".gz"
// sibling when the client accepts gzip. Bulma's stylesheet is ~678 KB raw but
// ~64 KB gzipped, and Go's http.FileServer does no compression of its own.
type staticHandler struct {
	files    fs.FS
	fallback http.Handler
}

func (h *staticHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", staticCacheControl)

	name, ok := h.gzipCandidate(r)
	if !ok {
		h.fallback.ServeHTTP(w, r)
		return
	}

	data, err := fs.ReadFile(h.files, name+".gz")
	if err != nil {
		// No pre-compressed sibling: serve the plain file.
		h.fallback.ServeHTTP(w, r)
		return
	}

	// Content-Type must reflect the *decoded* file, not the gzip wrapper.
	// Setting it explicitly also stops ServeContent from sniffing.
	if ctype := mime.TypeByExtension(path.Ext(name)); ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Add("Vary", "Accept-Encoding")

	// The content is embedded at compile time, so there is no meaningful
	// modification time; a zero time makes ServeContent skip Last-Modified.
	http.ServeContent(w, r, name, time.Time{}, bytes.NewReader(data))
}

// gzipCandidate returns the cleaned asset path when the request is eligible for
// a pre-compressed response, i.e. a plain GET/HEAD whose client accepts gzip.
func (h *staticHandler) gzipCandidate(r *http.Request) (string, bool) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return "", false
	}
	if !acceptsGzip(r.Header.Get("Accept-Encoding")) {
		return "", false
	}
	// Range requests can't be satisfied from a whole pre-compressed blob.
	if r.Header.Get("Range") != "" {
		return "", false
	}

	name := path.Clean(strings.TrimPrefix(r.URL.Path, "/static/"))
	if name == "." || name == "/" || strings.HasPrefix(name, "..") {
		return "", false
	}
	// Never serve a .gz as if it were the decoded asset.
	if strings.HasSuffix(name, ".gz") {
		return "", false
	}
	return name, true
}

// acceptsGzip reports whether an Accept-Encoding header allows gzip.
func acceptsGzip(header string) bool {
	for part := range strings.SplitSeq(header, ",") {
		enc, params, _ := strings.Cut(strings.TrimSpace(part), ";")
		if !strings.EqualFold(strings.TrimSpace(enc), "gzip") {
			continue
		}
		// "gzip;q=0" is an explicit refusal.
		if strings.ReplaceAll(strings.TrimSpace(params), " ", "") == "q=0" {
			return false
		}
		return true
	}
	return false
}
