package app

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestParsePaginationParams_FullRequestFlow tests the complete request parsing flow
func TestParsePaginationParams_FullRequestFlow(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		wantPage     int
		wantErr      bool
		description  string
	}{
		{
			name:        "Standard pagination request",
			url:         "/?folder=data/&page=5",
			wantPage:    5,
			wantErr:     false,
			description: "Normal pagination with folder parameter",
		},
		{
			name:        "First page explicit",
			url:         "/?folder=logs/2024/&page=1",
			wantPage:    1,
			wantErr:     false,
			description: "Explicit page 1",
		},
		{
			name:        "First page implicit (no page param)",
			url:         "/?folder=images/",
			wantPage:    1,
			wantErr:     false,
			description: "Defaults to page 1 when not specified",
		},
		{
			name:        "Deep folder structure with page",
			url:         "/?folder=a/b/c/d/e/&page=3",
			wantPage:    3,
			wantErr:     false,
			description: "Deeply nested folder with pagination",
		},
		{
			name:        "Special characters in folder",
			url:         "/?folder=" + url.QueryEscape("data with spaces/") + "&page=2",
			wantPage:    2,
			wantErr:     false,
			description: "Folder with spaces (URL encoded)",
		},
		{
			name:        "Empty folder (root)",
			url:         "/?folder=&page=1",
			wantPage:    1,
			wantErr:     false,
			description: "Root folder listing",
		},
		{
			name:        "Invalid page zero",
			url:         "/?folder=test/&page=0",
			wantPage:    0,
			wantErr:     true,
			description: "Page 0 is invalid",
		},
		{
			name:        "Invalid negative page",
			url:         "/?folder=test/&page=-1",
			wantPage:    0,
			wantErr:     true,
			description: "Negative pages are invalid",
		},
		{
			name:        "Invalid non-numeric page",
			url:         "/?folder=test/&page=abc",
			wantPage:    0,
			wantErr:     true,
			description: "Non-numeric page parameter",
		},
		{
			name:        "Very large page number",
			url:         "/?folder=test/&page=999999",
			wantPage:    999999,
			wantErr:     false,
			description: "Large page numbers are parsed correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			page, err := ParsePaginationParams(req)

			if tt.wantErr {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}

			assert.Equal(t, tt.wantPage, page, tt.description)
		})
	}
}

// TestValidatePageNumber_IntegrationScenarios tests realistic pagination scenarios
func TestValidatePageNumber_IntegrationScenarios(t *testing.T) {
	tests := []struct {
		name            string
		requestedPage   int
		totalPages      int
		expectedPage    int
		shouldRedirect  bool
		description     string
	}{
		{
			name:           "User requests page 5 of 10 - valid",
			requestedPage:  5,
			totalPages:     10,
			expectedPage:   5,
			shouldRedirect: false,
			description:    "Valid page in middle of range",
		},
		{
			name:           "User requests page 1 - always valid",
			requestedPage:  1,
			totalPages:     10,
			expectedPage:   1,
			shouldRedirect: false,
			description:    "First page is always valid",
		},
		{
			name:           "User requests last page",
			requestedPage:  10,
			totalPages:     10,
			expectedPage:   10,
			shouldRedirect: false,
			description:    "Last page is valid",
		},
		{
			name:           "User requests page beyond total - redirect to page 1",
			requestedPage:  15,
			totalPages:     10,
			expectedPage:   1,
			shouldRedirect: true,
			description:    "Page above max redirects to page 1",
		},
		{
			name:           "User bookmarked old page that no longer exists",
			requestedPage:  50,
			totalPages:     5,
			expectedPage:   1,
			shouldRedirect: true,
			description:    "Stale bookmark redirects to page 1",
		},
		{
			name:           "Folder is empty (1 page total)",
			requestedPage:  1,
			totalPages:     1,
			expectedPage:   1,
			shouldRedirect: false,
			description:    "Single page scenario",
		},
		{
			name:           "User somehow requests page 0",
			requestedPage:  0,
			totalPages:     10,
			expectedPage:   1,
			shouldRedirect: true,
			description:    "Page 0 redirects to page 1",
		},
		{
			name:           "Negative page from malformed request",
			requestedPage:  -5,
			totalPages:     10,
			expectedPage:   1,
			shouldRedirect: true,
			description:    "Negative page redirects to page 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validatedPage := ValidatePageNumber(tt.requestedPage, tt.totalPages)

			assert.Equal(t, tt.expectedPage, validatedPage, tt.description)

			// Check if redirect would be needed
			actualRedirect := (validatedPage != tt.requestedPage)
			assert.Equal(t, tt.shouldRedirect, actualRedirect,
				"Redirect expectation mismatch: %s", tt.description)
		})
	}
}

// TestPaginationURLConstruction tests that pagination URLs are constructed correctly
func TestPaginationURLConstruction(t *testing.T) {
	tests := []struct {
		name         string
		folder       string
		page         int
		expectedURL  string
		description  string
	}{
		{
			name:        "Root folder, page 1",
			folder:      "",
			page:        1,
			expectedURL: "/?folder=&page=1",
			description: "Root folder URL construction",
		},
		{
			name:        "Simple folder, page 2",
			folder:      "data/",
			page:        2,
			expectedURL: "/?folder=data/&page=2",
			description: "Simple folder pagination",
		},
		{
			name:        "Nested folder, page 5",
			folder:      "logs/2024/01/",
			page:        5,
			expectedURL: "/?folder=logs/2024/01/&page=5",
			description: "Nested folder pagination",
		},
		{
			name:        "Folder with spaces",
			folder:      "my documents/",
			page:        1,
			expectedURL: "/?folder=my+documents%2F&page=1", // URL encoded
			description: "Special characters in folder name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Construct URL as the template would
			constructedURL := fmt.Sprintf("/?folder=%s&page=%d",
				url.QueryEscape(tt.folder), tt.page)

			// Parse both URLs to compare them properly
			parsedExpected, err := url.Parse(tt.expectedURL)
			assert.NoError(t, err)

			parsedConstructed, err := url.Parse(constructedURL)
			assert.NoError(t, err)

			// Compare query parameters
			assert.Equal(t, parsedExpected.Query().Get("folder"),
				parsedConstructed.Query().Get("folder"),
				"Folder parameter mismatch: %s", tt.description)

			assert.Equal(t, parsedExpected.Query().Get("page"),
				parsedConstructed.Query().Get("page"),
				"Page parameter mismatch: %s", tt.description)
		})
	}
}

// TestPaginationEdgeCases tests edge cases and boundary conditions
func TestPaginationEdgeCases(t *testing.T) {
	t.Run("Concurrent page requests don't interfere", func(t *testing.T) {
		// Simulate multiple simultaneous pagination requests
		pages := []int{1, 2, 3, 4, 5}
		results := make(chan int, len(pages))

		for _, page := range pages {
			go func(p int) {
				req := httptest.NewRequest(http.MethodGet,
					fmt.Sprintf("/?page=%d", p), nil)
				parsedPage, _ := ParsePaginationParams(req)
				results <- parsedPage
			}(page)
		}

		// Collect results
		collected := make(map[int]bool)
		for i := 0; i < len(pages); i++ {
			result := <-results
			collected[result] = true
		}

		// Verify all pages were parsed correctly
		for _, page := range pages {
			assert.True(t, collected[page],
				"Page %d should have been parsed", page)
		}
	})

	t.Run("Empty query string defaults correctly", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		page, err := ParsePaginationParams(req)

		assert.NoError(t, err)
		assert.Equal(t, 1, page, "Should default to page 1")
	})

	t.Run("Multiple page parameters uses first value", func(t *testing.T) {
		// URL with duplicate page parameters
		req := httptest.NewRequest(http.MethodGet, "/?page=2&page=3", nil)
		page, err := ParsePaginationParams(req)

		assert.NoError(t, err)
		assert.Equal(t, 2, page, "Should use first page parameter")
	})

	t.Run("Page parameter with whitespace", func(t *testing.T) {
		// Test various whitespace scenarios
		testCases := []struct {
			url     string
			wantErr bool
		}{
			{"/?page=%201%20", true},  // Spaces around number
			{"/?page=2%20", true},      // Trailing space
			{"/?page=%202", true},      // Leading space
		}

		for _, tc := range testCases {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			_, err := ParsePaginationParams(req)

			if tc.wantErr {
				assert.Error(t, err, "URL: %s", tc.url)
			}
		}
	})
}

// TestValidatePageNumber_PerformanceScenario simulates realistic load
func TestValidatePageNumber_PerformanceScenario(t *testing.T) {
	// Simulate validating 10,000 page requests
	totalPages := 100
	iterations := 10000

	for i := 0; i < iterations; i++ {
		// Randomly generate page requests (1-150 range)
		requestedPage := (i % 150) + 1
		validPage := ValidatePageNumber(requestedPage, totalPages)

		// Verify the result is always valid
		assert.True(t, validPage >= 1 && validPage <= totalPages,
			"Page %d should be between 1 and %d, got %d",
			requestedPage, totalPages, validPage)
	}
}

// Note: Full end-to-end integration tests would require:
// 1. Starting the HTTP server with a test database
// 2. Making real HTTP requests via http.Client
// 3. Verifying HTML responses contain correct pagination controls
// 4. Testing HTMX interactions and partial page updates
// 5. Testing with actual S3 data and database state
//
// Example structure for future E2E tests:
//
// func TestPaginationEndToEnd(t *testing.T) {
//     if testing.Short() {
//         t.Skip("Skipping E2E test")
//     }
//
//     // Setup test database and S3 mock
//     testDB := setupTestDatabase(t)
//     defer testDB.Close()
//
//     // Seed database with 500 objects
//     seedData(t, testDB, "test-bucket", "test/", 500)
//
//     // Start HTTP server
//     server := startTestServer(t, testDB)
//     defer server.Close()
//
//     // Test: Request page 1
//     resp, err := http.Get(server.URL + "/?folder=test/&page=1")
//     require.NoError(t, err)
//     assert.Equal(t, 200, resp.StatusCode)
//
//     // Verify response contains exactly 50 items
//     body := readBody(t, resp)
//     assert.Contains(t, body, "Page 1 of 10")
//     assert.Contains(t, body, "500 items")
//
//     // Test: Request invalid page redirects
//     resp, err = http.Get(server.URL + "/?folder=test/&page=99")
//     require.NoError(t, err)
//     assert.Equal(t, 302, resp.StatusCode) // Redirect
//     assert.Contains(t, resp.Header.Get("Location"), "page=1")
// }
