package app

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

var (
	// ErrInvalidPageFormat is returned when the page parameter cannot be parsed as a number.
	ErrInvalidPageFormat = errors.New("invalid page parameter: must be a number")

	// ErrInvalidPageValue is returned when the page parameter is less than 1.
	ErrInvalidPageValue = errors.New("invalid page parameter: must be >= 1")
)

// ParsePaginationParams extracts and validates the page number from HTTP request query parameters.
// It returns the page number (1-indexed) or an error if parsing fails.
//
// Parameters:
//   - r: HTTP request containing query parameters
//
// Returns:
//   - page: Page number (defaults to 1 if not specified or invalid)
//   - error: Error if the parameter exists but cannot be parsed or is invalid
//
// Behavior:
//   - Missing parameter: Returns page=1, no error
//   - Empty parameter: Returns page=1, no error
//   - Valid number >= 1: Returns the number, no error
//   - Invalid format (non-numeric): Returns 0, error
//   - Number < 1: Returns 0, error
func ParsePaginationParams(r *http.Request) (int, error) {
	pageStr := r.URL.Query().Get("page")

	// Default to page 1 if not specified
	if pageStr == "" {
		return 1, nil
	}

	// Parse the page number
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrInvalidPageFormat, err)
	}

	// Validate page number is positive
	if page < 1 {
		return 0, ErrInvalidPageValue
	}

	return page, nil
}

// ValidatePageNumber ensures a page number is within valid bounds.
// It returns a safe page number, auto-correcting out-of-bounds values to 1.
//
// Parameters:
//   - page: Requested page number
//   - maxPages: Maximum valid page number (must be >= 1)
//
// Returns:
//   - Corrected page number (1 if out of bounds, otherwise the original page)
//
// Use cases:
//   - Before redirects: Ensures users don't land on invalid pages
//   - After database queries: Validates against actual data bounds
func ValidatePageNumber(page, maxPages int) int {
	// Auto-correct to page 1 if out of valid range
	if page < 1 || page > maxPages {
		return 1
	}
	return page
}
