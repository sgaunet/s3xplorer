package views

import (
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
			name:            "Serve app.css",
			path:            "/static/app.css",
			expectedStatus:  http.StatusOK,
			contentContains: "tailwindcss",
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
	req := httptest.NewRequest(http.MethodGet, "/static/app.css", nil)
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
}
