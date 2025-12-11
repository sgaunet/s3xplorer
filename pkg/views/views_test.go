package views

import (
	"io/fs"
	"strings"
	"testing"
)

// TestEmbeddedStaticAssets verifies that all required static assets are embedded
// and have valid content.
func TestEmbeddedStaticAssets(t *testing.T) {
	// Define required assets
	requiredAssets := []string{
		"static/app.css",
		"static/app.js",
		"static/icons.svg",
		"static/file-heart.png",
	}

	// Test 1: Verify all required files are present
	t.Run("AllRequiredFilesPresent", func(t *testing.T) {
		for _, assetPath := range requiredAssets {
			_, err := fs.Stat(staticCSS, assetPath)
			if err != nil {
				t.Errorf("Required asset %s not found in embed.FS: %v", assetPath, err)
			}
		}
	})

	// Test 2: Verify all files have non-zero size
	t.Run("AllFilesHaveNonZeroSize", func(t *testing.T) {
		for _, assetPath := range requiredAssets {
			data, err := fs.ReadFile(staticCSS, assetPath)
			if err != nil {
				t.Errorf("Failed to read %s: %v", assetPath, err)
				continue
			}
			if len(data) == 0 {
				t.Errorf("Asset %s has zero size", assetPath)
			}
		}
	})

	// Test 3: Verify app.css contains Tailwind CSS
	t.Run("AppCSSContainsTailwind", func(t *testing.T) {
		data, err := fs.ReadFile(staticCSS, "static/app.css")
		if err != nil {
			t.Fatalf("Failed to read app.css: %v", err)
		}

		content := string(data)
		// Tailwind CSS includes this in the generated output
		if !strings.Contains(content, "tailwindcss") && !strings.Contains(content, "Tailwind CSS") {
			t.Error("app.css does not appear to contain Tailwind CSS content")
		}
	})

	// Test 4: Verify static directory structure
	t.Run("StaticDirectoryStructure", func(t *testing.T) {
		entries, err := fs.ReadDir(staticCSS, "static")
		if err != nil {
			t.Fatalf("Failed to read static directory: %v", err)
		}

		// Should have at least 4 files (app.css, app.js, icons.svg, file-heart.png)
		// May have more if src/ directory is present
		if len(entries) < 4 {
			t.Errorf("Expected at least 4 entries in static directory, got %d", len(entries))
		}

		// Verify specific files exist in directory listing
		fileMap := make(map[string]bool)
		for _, entry := range entries {
			fileMap[entry.Name()] = true
		}

		expectedFiles := []string{"app.css", "app.js", "icons.svg", "file-heart.png"}
		for _, expectedFile := range expectedFiles {
			if !fileMap[expectedFile] {
				t.Errorf("Expected file %s not found in static directory listing", expectedFile)
			}
		}
	})

	// Test 5: Verify favicon is embedded separately
	t.Run("FaviconEmbedded", func(t *testing.T) {
		if len(faviconFS) == 0 {
			t.Error("Favicon (file-heart.png) has zero size in faviconFS")
		}
	})
}
