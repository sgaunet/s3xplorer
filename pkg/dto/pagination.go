package dto

// PaginationInfo holds pagination metadata for paginated results.
// All page numbers are 1-indexed (first page is 1), while StartIndex and EndIndex
// are 0-indexed positions for array/slice operations.
type PaginationInfo struct {
	// CurrentPage is the current page number (1-indexed).
	CurrentPage int `json:"currentPage"`

	// TotalPages is the total number of pages available.
	TotalPages int `json:"totalPages"`

	// TotalItems is the total number of items across all pages.
	TotalItems int64 `json:"totalItems"`

	// PageSize is the maximum number of items per page.
	PageSize int `json:"pageSize"`

	// HasPrevious indicates if there is a previous page available.
	HasPrevious bool `json:"hasPrevious"`

	// HasNext indicates if there is a next page available.
	HasNext bool `json:"hasNext"`

	// StartIndex is the 0-indexed position of the first item on this page
	// in the complete result set. Use this for array/slice operations.
	StartIndex int `json:"startIndex"`

	// EndIndex is the 0-indexed position (exclusive) of the last item on this page
	// in the complete result set. Use this for array/slice operations like items[StartIndex:EndIndex].
	EndIndex int `json:"endIndex"`
}

// NewPaginationInfo creates a new PaginationInfo instance and calculates all derived fields.
// Parameters:
//   - totalItems: Total number of items across all pages
//   - pageSize: Maximum number of items per page
//   - currentPage: Current page number (1-indexed)
//
// Returns a PaginationInfo with all fields properly calculated.
// Special case: When totalItems is 0, totalPages is set to 1 (not 0).
func NewPaginationInfo(totalItems int64, pageSize, currentPage int) PaginationInfo {
	// Calculate total pages using ceiling division
	// Formula: (totalItems + pageSize - 1) / pageSize
	totalPages := 1
	if totalItems > 0 && pageSize > 0 {
		totalPages = int((totalItems + int64(pageSize) - 1) / int64(pageSize))
	}

	// Ensure currentPage is within valid range
	if currentPage < 1 {
		currentPage = 1
	}
	if currentPage > totalPages {
		currentPage = totalPages
	}

	// Calculate 0-indexed start position for array slicing
	startIndex := (currentPage - 1) * pageSize

	// Calculate 0-indexed end position (exclusive) for array slicing
	endIndex := min(startIndex+pageSize, int(totalItems))

	return PaginationInfo{
		CurrentPage: currentPage,
		TotalPages:  totalPages,
		TotalItems:  totalItems,
		PageSize:    pageSize,
		HasPrevious: currentPage > 1,
		HasNext:     currentPage < totalPages,
		StartIndex:  startIndex,
		EndIndex:    endIndex,
	}
}
