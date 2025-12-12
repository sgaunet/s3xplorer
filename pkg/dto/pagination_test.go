package dto

import "testing"

func TestNewPaginationInfo_ZeroItems(t *testing.T) {
	p := NewPaginationInfo(0, 50, 1)

	if p.TotalItems != 0 {
		t.Errorf("Expected TotalItems=0, got %d", p.TotalItems)
	}
	if p.TotalPages != 1 {
		t.Errorf("Expected TotalPages=1 for zero items, got %d", p.TotalPages)
	}
	if p.CurrentPage != 1 {
		t.Errorf("Expected CurrentPage=1, got %d", p.CurrentPage)
	}
	if p.HasPrevious {
		t.Error("Expected HasPrevious=false for page 1")
	}
	if p.HasNext {
		t.Error("Expected HasNext=false for single page")
	}
	if p.StartIndex != 0 {
		t.Errorf("Expected StartIndex=0, got %d", p.StartIndex)
	}
	if p.EndIndex != 0 {
		t.Errorf("Expected EndIndex=0, got %d", p.EndIndex)
	}
}

func TestNewPaginationInfo_SinglePageLessThan50(t *testing.T) {
	p := NewPaginationInfo(25, 50, 1)

	if p.TotalItems != 25 {
		t.Errorf("Expected TotalItems=25, got %d", p.TotalItems)
	}
	if p.TotalPages != 1 {
		t.Errorf("Expected TotalPages=1, got %d", p.TotalPages)
	}
	if p.CurrentPage != 1 {
		t.Errorf("Expected CurrentPage=1, got %d", p.CurrentPage)
	}
	if p.HasPrevious {
		t.Error("Expected HasPrevious=false for page 1")
	}
	if p.HasNext {
		t.Error("Expected HasNext=false for single page")
	}
	if p.StartIndex != 0 {
		t.Errorf("Expected StartIndex=0, got %d", p.StartIndex)
	}
	if p.EndIndex != 25 {
		t.Errorf("Expected EndIndex=25, got %d", p.EndIndex)
	}
}

func TestNewPaginationInfo_ExactlyOnePageOf50(t *testing.T) {
	p := NewPaginationInfo(50, 50, 1)

	if p.TotalItems != 50 {
		t.Errorf("Expected TotalItems=50, got %d", p.TotalItems)
	}
	if p.TotalPages != 1 {
		t.Errorf("Expected TotalPages=1, got %d", p.TotalPages)
	}
	if p.CurrentPage != 1 {
		t.Errorf("Expected CurrentPage=1, got %d", p.CurrentPage)
	}
	if p.HasPrevious {
		t.Error("Expected HasPrevious=false for page 1")
	}
	if p.HasNext {
		t.Error("Expected HasNext=false for single page")
	}
	if p.StartIndex != 0 {
		t.Errorf("Expected StartIndex=0, got %d", p.StartIndex)
	}
	if p.EndIndex != 50 {
		t.Errorf("Expected EndIndex=50, got %d", p.EndIndex)
	}
}

func TestNewPaginationInfo_TwoPages_FirstPage(t *testing.T) {
	p := NewPaginationInfo(100, 50, 1)

	if p.TotalItems != 100 {
		t.Errorf("Expected TotalItems=100, got %d", p.TotalItems)
	}
	if p.TotalPages != 2 {
		t.Errorf("Expected TotalPages=2, got %d", p.TotalPages)
	}
	if p.CurrentPage != 1 {
		t.Errorf("Expected CurrentPage=1, got %d", p.CurrentPage)
	}
	if p.HasPrevious {
		t.Error("Expected HasPrevious=false for page 1")
	}
	if !p.HasNext {
		t.Error("Expected HasNext=true for page 1 of 2")
	}
	if p.StartIndex != 0 {
		t.Errorf("Expected StartIndex=0, got %d", p.StartIndex)
	}
	if p.EndIndex != 50 {
		t.Errorf("Expected EndIndex=50, got %d", p.EndIndex)
	}
}

func TestNewPaginationInfo_TwoPages_SecondPage(t *testing.T) {
	p := NewPaginationInfo(100, 50, 2)

	if p.TotalItems != 100 {
		t.Errorf("Expected TotalItems=100, got %d", p.TotalItems)
	}
	if p.TotalPages != 2 {
		t.Errorf("Expected TotalPages=2, got %d", p.TotalPages)
	}
	if p.CurrentPage != 2 {
		t.Errorf("Expected CurrentPage=2, got %d", p.CurrentPage)
	}
	if !p.HasPrevious {
		t.Error("Expected HasPrevious=true for page 2")
	}
	if p.HasNext {
		t.Error("Expected HasNext=false for last page")
	}
	if p.StartIndex != 50 {
		t.Errorf("Expected StartIndex=50, got %d", p.StartIndex)
	}
	if p.EndIndex != 100 {
		t.Errorf("Expected EndIndex=100, got %d", p.EndIndex)
	}
}

func TestNewPaginationInfo_FifteenPages(t *testing.T) {
	// 720 items / 50 per page = 14.4, rounds up to 15 pages
	p := NewPaginationInfo(720, 50, 1)

	if p.TotalItems != 720 {
		t.Errorf("Expected TotalItems=720, got %d", p.TotalItems)
	}
	if p.TotalPages != 15 {
		t.Errorf("Expected TotalPages=15, got %d", p.TotalPages)
	}
	if p.CurrentPage != 1 {
		t.Errorf("Expected CurrentPage=1, got %d", p.CurrentPage)
	}
	if p.HasPrevious {
		t.Error("Expected HasPrevious=false for page 1")
	}
	if !p.HasNext {
		t.Error("Expected HasNext=true for page 1 of 15")
	}
}

func TestNewPaginationInfo_FifteenPages_MiddlePage(t *testing.T) {
	p := NewPaginationInfo(720, 50, 8)

	if p.CurrentPage != 8 {
		t.Errorf("Expected CurrentPage=8, got %d", p.CurrentPage)
	}
	if !p.HasPrevious {
		t.Error("Expected HasPrevious=true for page 8")
	}
	if !p.HasNext {
		t.Error("Expected HasNext=true for page 8 of 15")
	}
	if p.StartIndex != 350 {
		t.Errorf("Expected StartIndex=350 (7*50), got %d", p.StartIndex)
	}
	if p.EndIndex != 400 {
		t.Errorf("Expected EndIndex=400 (8*50), got %d", p.EndIndex)
	}
}

func TestNewPaginationInfo_FifteenPages_LastPage(t *testing.T) {
	p := NewPaginationInfo(720, 50, 15)

	if p.CurrentPage != 15 {
		t.Errorf("Expected CurrentPage=15, got %d", p.CurrentPage)
	}
	if !p.HasPrevious {
		t.Error("Expected HasPrevious=true for page 15")
	}
	if p.HasNext {
		t.Error("Expected HasNext=false for last page")
	}
	if p.StartIndex != 700 {
		t.Errorf("Expected StartIndex=700 (14*50), got %d", p.StartIndex)
	}
	if p.EndIndex != 720 {
		t.Errorf("Expected EndIndex=720, got %d", p.EndIndex)
	}
}

func TestNewPaginationInfo_PartialLastPage(t *testing.T) {
	// 103 items / 50 per page = 2.06, rounds up to 3 pages
	// Last page has only 3 items
	p := NewPaginationInfo(103, 50, 3)

	if p.TotalPages != 3 {
		t.Errorf("Expected TotalPages=3, got %d", p.TotalPages)
	}
	if p.CurrentPage != 3 {
		t.Errorf("Expected CurrentPage=3, got %d", p.CurrentPage)
	}
	if p.StartIndex != 100 {
		t.Errorf("Expected StartIndex=100, got %d", p.StartIndex)
	}
	if p.EndIndex != 103 {
		t.Errorf("Expected EndIndex=103, got %d", p.EndIndex)
	}
}

func TestNewPaginationInfo_InvalidPageNumber(t *testing.T) {
	// Test page number < 1 (should default to 1)
	p := NewPaginationInfo(100, 50, 0)
	if p.CurrentPage != 1 {
		t.Errorf("Expected CurrentPage=1 when given 0, got %d", p.CurrentPage)
	}

	p = NewPaginationInfo(100, 50, -5)
	if p.CurrentPage != 1 {
		t.Errorf("Expected CurrentPage=1 when given -5, got %d", p.CurrentPage)
	}

	// Test page number > totalPages (should cap at totalPages)
	p = NewPaginationInfo(100, 50, 10)
	if p.CurrentPage != 2 {
		t.Errorf("Expected CurrentPage=2 (max pages) when given 10, got %d", p.CurrentPage)
	}
}

func TestNewPaginationInfo_EdgeCaseIndexCalculations(t *testing.T) {
	tests := []struct {
		name           string
		totalItems     int64
		pageSize       int
		currentPage    int
		wantStartIndex int
		wantEndIndex   int
	}{
		{"Page 1 of 100 items", 100, 50, 1, 0, 50},
		{"Page 2 of 100 items", 100, 50, 2, 50, 100},
		{"Page 1 of 25 items", 25, 50, 1, 0, 25},
		{"Page 3 of 103 items", 103, 50, 3, 100, 103},
		{"Page 15 of 720 items", 720, 50, 15, 700, 720},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPaginationInfo(tt.totalItems, tt.pageSize, tt.currentPage)
			if p.StartIndex != tt.wantStartIndex {
				t.Errorf("StartIndex = %d, want %d", p.StartIndex, tt.wantStartIndex)
			}
			if p.EndIndex != tt.wantEndIndex {
				t.Errorf("EndIndex = %d, want %d", p.EndIndex, tt.wantEndIndex)
			}
		})
	}
}
