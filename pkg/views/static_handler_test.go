package views

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestStaticHandler verifies that the StaticHandler serves embedded assets correctly.
func TestStaticHandler(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		expectedStatus  int
		contentContains string
		contentType     string
	}{
		{
			name:            "Serve bulma.min.css",
			path:            "/static/bulma.min.css",
			expectedStatus:  http.StatusOK,
			contentContains: "--bulma-scheme-main",
			contentType:     "text/css",
		},
		{
			name:            "Serve theme.css",
			path:            "/static/theme.css",
			expectedStatus:  http.StatusOK,
			contentContains: ".skip-link",
			contentType:     "text/css",
		},
		{
			name:            "Serve app.js",
			path:            "/static/app.js",
			expectedStatus:  http.StatusOK,
			contentContains: "toggleTheme",
			contentType:     "text/javascript",
		},
		{
			name:            "Serve icons.svg",
			path:            "/static/icons.svg",
			expectedStatus:  http.StatusOK,
			contentContains: "<svg",
			contentType:     "image/svg+xml",
		},
		{
			name:           "Serve file-heart.png",
			path:           "/static/file-heart.png",
			expectedStatus: http.StatusOK,
			contentType:    "image/png",
		},
		{
			name:           "Non-existent file returns 404",
			path:           "/static/nonexistent.css",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a test request
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			// Serve the request
			StaticHandler.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// For successful requests, verify content
			if tt.expectedStatus == http.StatusOK {
				// Verify response has content first
				if w.Body.Len() == 0 {
					t.Error("Response body is empty")
				}

				// Check content type if specified
				if tt.contentType != "" {
					contentType := w.Header().Get("Content-Type")
					if !strings.Contains(contentType, tt.contentType) {
						t.Errorf("Expected content type to contain %s, got %s", tt.contentType, contentType)
					}
				}

				// Check content contains expected string if specified
				if tt.contentContains != "" {
					body := w.Body.Bytes()
					bodyStr := string(body)
					if !strings.Contains(bodyStr, tt.contentContains) {
						t.Errorf("Expected response to contain %q, but it doesn't", tt.contentContains)
					}
				}
			}
		})
	}
}

// TestStaticHandlerBasicHeaders verifies that basic HTTP headers are set correctly.
func TestStaticHandlerBasicHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/static/bulma.min.css", nil)
	w := httptest.NewRecorder()

	StaticHandler.ServeHTTP(w, req)

	// Verify Content-Type header is set correctly
	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/css") {
		t.Errorf("Expected Content-Type to contain text/css, got %s", contentType)
	}

	// Verify Content-Length header is set
	contentLength := w.Header().Get("Content-Length")
	if contentLength == "" {
		t.Error("Expected Content-Length header to be set")
	}

	// Verify Accept-Ranges header is set (standard for file server)
	acceptRanges := w.Header().Get("Accept-Ranges")
	if acceptRanges != "bytes" {
		t.Errorf("Expected Accept-Ranges: bytes, got %s", acceptRanges)
	}

	// Static assets are cache-busted via ?v=N, so they may be cached forever.
	if cc := w.Header().Get("Cache-Control"); cc != staticCacheControl {
		t.Errorf("Expected Cache-Control %q, got %q", staticCacheControl, cc)
	}
}

// TestStaticHandlerGzip verifies that a pre-compressed sibling is served when
// the client accepts gzip, and that the plain file is served otherwise.
func TestStaticHandlerGzip(t *testing.T) {
	t.Run("ServesGzipWhenAccepted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/bulma.min.css", nil)
		req.Header.Set("Accept-Encoding", "gzip, deflate, br")
		w := httptest.NewRecorder()

		StaticHandler.ServeHTTP(w, req)

		if got := w.Header().Get("Content-Encoding"); got != "gzip" {
			t.Errorf("Expected Content-Encoding gzip, got %q", got)
		}
		if !strings.Contains(w.Header().Get("Content-Type"), "text/css") {
			t.Errorf("Expected the decoded Content-Type, got %q", w.Header().Get("Content-Type"))
		}
		if !strings.Contains(w.Header().Get("Vary"), "Accept-Encoding") {
			t.Error("Expected Vary to include Accept-Encoding")
		}

		// The body must be real gzip, and smaller than the raw stylesheet.
		body := w.Body.Bytes()
		if len(body) < 2 || body[0] != 0x1f || body[1] != 0x8b {
			t.Fatal("Response body is not gzip-encoded")
		}
		gz, err := gzip.NewReader(bytes.NewReader(body))
		if err != nil {
			t.Fatalf("Failed to open gzip reader: %v", err)
		}
		defer func() { _ = gz.Close() }()
		decoded, err := io.ReadAll(gz)
		if err != nil {
			t.Fatalf("Failed to decompress body: %v", err)
		}
		if !strings.Contains(string(decoded), "--bulma-scheme-main") {
			t.Error("Decompressed body is not the Bulma stylesheet")
		}
		if len(body) >= len(decoded) {
			t.Errorf("Gzip body (%d) is not smaller than raw (%d)", len(body), len(decoded))
		}
	})

	t.Run("ServesPlainWhenGzipNotAccepted", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/bulma.min.css", nil)
		w := httptest.NewRecorder()

		StaticHandler.ServeHTTP(w, req)

		if got := w.Header().Get("Content-Encoding"); got != "" {
			t.Errorf("Expected no Content-Encoding, got %q", got)
		}
		if !strings.Contains(w.Body.String(), "--bulma-scheme-main") {
			t.Error("Expected the plain Bulma stylesheet")
		}
	})

	t.Run("ServesPlainWhenGzipRefused", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/static/bulma.min.css", nil)
		req.Header.Set("Accept-Encoding", "gzip;q=0")
		w := httptest.NewRecorder()

		StaticHandler.ServeHTTP(w, req)

		if got := w.Header().Get("Content-Encoding"); got != "" {
			t.Errorf("Expected no Content-Encoding for gzip;q=0, got %q", got)
		}
	})

	t.Run("FallsBackWhenNoGzSibling", func(t *testing.T) {
		// icons.svg has no pre-compressed sibling.
		req := httptest.NewRequest(http.MethodGet, "/static/icons.svg", nil)
		req.Header.Set("Accept-Encoding", "gzip")
		w := httptest.NewRecorder()

		StaticHandler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected 200, got %d", w.Code)
		}
		if got := w.Header().Get("Content-Encoding"); got != "" {
			t.Errorf("Expected no Content-Encoding, got %q", got)
		}
		if !strings.Contains(w.Body.String(), "<svg") {
			t.Error("Expected the icon sprite")
		}
	})
}
