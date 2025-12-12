package app

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParsePaginationParams(t *testing.T) {
	tests := []struct {
		name      string
		queryURL  string
		wantPage  int
		wantError bool
	}{
		// Valid inputs
		{
			name:      "Valid page 1",
			queryURL:  "/?page=1",
			wantPage:  1,
			wantError: false,
		},
		{
			name:      "Valid page 5",
			queryURL:  "/?page=5",
			wantPage:  5,
			wantError: false,
		},
		{
			name:      "Valid page 100",
			queryURL:  "/?page=100",
			wantPage:  100,
			wantError: false,
		},
		{
			name:      "Valid large page number",
			queryURL:  "/?page=9999",
			wantPage:  9999,
			wantError: false,
		},
		// Missing parameter - defaults to 1
		{
			name:      "Missing page parameter",
			queryURL:  "/",
			wantPage:  1,
			wantError: false,
		},
		{
			name:      "Empty page parameter",
			queryURL:  "/?page=",
			wantPage:  1,
			wantError: false,
		},
		// Invalid inputs - should return error
		{
			name:      "Page zero",
			queryURL:  "/?page=0",
			wantPage:  0,
			wantError: true,
		},
		{
			name:      "Negative page",
			queryURL:  "/?page=-5",
			wantPage:  0,
			wantError: true,
		},
		{
			name:      "Non-numeric page",
			queryURL:  "/?page=abc",
			wantPage:  0,
			wantError: true,
		},
		{
			name:      "Decimal page number",
			queryURL:  "/?page=1.5",
			wantPage:  0,
			wantError: true,
		},
		{
			name:      "Page with special characters",
			queryURL:  "/?page=1@",
			wantPage:  0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.queryURL, nil)
			gotPage, err := ParsePaginationParams(req)

			if tt.wantError && err == nil {
				t.Errorf("ParsePaginationParams() expected error but got none")
			}
			if !tt.wantError && err != nil {
				t.Errorf("ParsePaginationParams() unexpected error: %v", err)
			}
			if gotPage != tt.wantPage {
				t.Errorf("ParsePaginationParams() = %d, want %d", gotPage, tt.wantPage)
			}
		})
	}
}

func TestValidatePageNumber(t *testing.T) {
	tests := []struct {
		name     string
		page     int
		maxPages int
		want     int
	}{
		// Valid range
		{
			name:     "Page within range (middle)",
			page:     3,
			maxPages: 10,
			want:     3,
		},
		{
			name:     "Page at start of range",
			page:     1,
			maxPages: 10,
			want:     1,
		},
		{
			name:     "Page at end of range",
			page:     10,
			maxPages: 10,
			want:     10,
		},
		{
			name:     "Single page scenario",
			page:     1,
			maxPages: 1,
			want:     1,
		},
		// Below range - should return 1
		{
			name:     "Page zero",
			page:     0,
			maxPages: 10,
			want:     1,
		},
		{
			name:     "Negative page",
			page:     -5,
			maxPages: 10,
			want:     1,
		},
		{
			name:     "Very negative page",
			page:     -999,
			maxPages: 10,
			want:     1,
		},
		// Above range - should return 1
		{
			name:     "Page above max",
			page:     15,
			maxPages: 10,
			want:     1,
		},
		{
			name:     "Page far above max",
			page:     100,
			maxPages: 10,
			want:     1,
		},
		{
			name:     "Page one above max",
			page:     11,
			maxPages: 10,
			want:     1,
		},
		// Edge cases
		{
			name:     "Large page numbers within range",
			page:     500,
			maxPages: 1000,
			want:     500,
		},
		{
			name:     "Max pages is 1, page is 2",
			page:     2,
			maxPages: 1,
			want:     1,
		},
		{
			name:     "Max pages is 1, page is 0",
			page:     0,
			maxPages: 1,
			want:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidatePageNumber(tt.page, tt.maxPages)
			if got != tt.want {
				t.Errorf("ValidatePageNumber(%d, %d) = %d, want %d", tt.page, tt.maxPages, got, tt.want)
			}
		})
	}
}

// TestParsePaginationParams_WithOtherQueryParams tests that pagination parsing
// works correctly when other query parameters are present
func TestParsePaginationParams_WithOtherQueryParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/?prefix=test/&page=5&sort=name", nil)
	page, err := ParsePaginationParams(req)

	if err != nil {
		t.Errorf("ParsePaginationParams() unexpected error: %v", err)
	}
	if page != 5 {
		t.Errorf("ParsePaginationParams() = %d, want 5", page)
	}
}

// TestValidatePageNumber_BoundaryConditions tests specific boundary scenarios
func TestValidatePageNumber_BoundaryConditions(t *testing.T) {
	// Test exact boundary at max
	if got := ValidatePageNumber(10, 10); got != 10 {
		t.Errorf("ValidatePageNumber(10, 10) = %d, want 10", got)
	}

	// Test one above max
	if got := ValidatePageNumber(11, 10); got != 1 {
		t.Errorf("ValidatePageNumber(11, 10) = %d, want 1", got)
	}

	// Test exact boundary at min
	if got := ValidatePageNumber(1, 10); got != 1 {
		t.Errorf("ValidatePageNumber(1, 10) = %d, want 1", got)
	}

	// Test one below min
	if got := ValidatePageNumber(0, 10); got != 1 {
		t.Errorf("ValidatePageNumber(0, 10) = %d, want 1", got)
	}
}
