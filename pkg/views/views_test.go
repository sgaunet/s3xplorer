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
		"static/bulma.min.css",
		"static/bulma.min.css.gz",
		"static/theme.css",
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

	// Test 3: Verify the vendored stylesheet really is Bulma
	t.Run("BulmaCSSPresent", func(t *testing.T) {
		data, err := fs.ReadFile(staticCSS, "static/bulma.min.css")
		if err != nil {
			t.Fatalf("Failed to read bulma.min.css: %v", err)
		}

		content := string(data)
		for _, want := range []string{"bulma.io", "--bulma-scheme-main", "[data-theme=dark]"} {
			if !strings.Contains(content, want) {
				t.Errorf("bulma.min.css does not contain %q", want)
			}
		}
	})

	// Test 3b: theme.css must size the sprite icons. Bulma sizes only the .icon
	// wrapper, so without this rule every icon renders at the 300x150 SVG default.
	t.Run("ThemeCSSSizesIcons", func(t *testing.T) {
		data, err := fs.ReadFile(staticCSS, "static/theme.css")
		if err != nil {
			t.Fatalf("Failed to read theme.css: %v", err)
		}

		content := string(data)
		for _, want := range []string{".icon > svg", "--s3x-icon", ".skip-link"} {
			if !strings.Contains(content, want) {
				t.Errorf("theme.css does not contain %q", want)
			}
		}
	})

	// Test 4: Verify static directory structure
	t.Run("StaticDirectoryStructure", func(t *testing.T) {
		entries, err := fs.ReadDir(staticCSS, "static")
		if err != nil {
			t.Fatalf("Failed to read static directory: %v", err)
		}

		// bulma.min.css, bulma.min.css.gz, theme.css, theme.css.gz, app.js,
		// icons.svg, file-heart.png
		if len(entries) < 7 {
			t.Errorf("Expected at least 7 entries in static directory, got %d", len(entries))
		}

		// Verify specific files exist in directory listing
		fileMap := make(map[string]bool)
		for _, entry := range entries {
			fileMap[entry.Name()] = true
		}

		expectedFiles := []string{
			"bulma.min.css", "bulma.min.css.gz", "theme.css",
			"app.js", "icons.svg", "file-heart.png",
		}
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
