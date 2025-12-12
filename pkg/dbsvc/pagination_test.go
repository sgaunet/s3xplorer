package dbsvc

import "testing"

// TestCalculateFolderFileOffsets_AllFolders tests pagination with only folders, no files
func TestCalculateFolderFileOffsets_AllFolders(t *testing.T) {
	const pageSize = 50
	totalFolders := int64(120)
	totalFiles := int64(0)

	tests := []struct {
		page                                                           int
		wantFolderLimit, wantFolderOffset, wantFileLimit, wantFileOffset int
	}{
		// Page 1: folders 0-49
		{page: 1, wantFolderLimit: 50, wantFolderOffset: 0, wantFileLimit: 0, wantFileOffset: 0},
		// Page 2: folders 50-99
		{page: 2, wantFolderLimit: 50, wantFolderOffset: 50, wantFileLimit: 0, wantFileOffset: 0},
		// Page 3: folders 100-119 (only 20 folders remaining)
		{page: 3, wantFolderLimit: 20, wantFolderOffset: 100, wantFileLimit: 0, wantFileOffset: 0},
	}

	for _, tt := range tests {
		t.Run("page_"+string(rune(tt.page+'0')), func(t *testing.T) {
			folderLimit, folderOffset, fileLimit, fileOffset := CalculateFolderFileOffsets(
				tt.page, pageSize, totalFolders, totalFiles,
			)

			if folderLimit != tt.wantFolderLimit {
				t.Errorf("folderLimit = %d, want %d", folderLimit, tt.wantFolderLimit)
			}
			if folderOffset != tt.wantFolderOffset {
				t.Errorf("folderOffset = %d, want %d", folderOffset, tt.wantFolderOffset)
			}
			if fileLimit != tt.wantFileLimit {
				t.Errorf("fileLimit = %d, want %d", fileLimit, tt.wantFileLimit)
			}
			if fileOffset != tt.wantFileOffset {
				t.Errorf("fileOffset = %d, want %d", fileOffset, tt.wantFileOffset)
			}
		})
	}
}

// TestCalculateFolderFileOffsets_AllFiles tests pagination with only files, no folders
func TestCalculateFolderFileOffsets_AllFiles(t *testing.T) {
	const pageSize = 50
	totalFolders := int64(0)
	totalFiles := int64(200)

	tests := []struct {
		page                                                           int
		wantFolderLimit, wantFolderOffset, wantFileLimit, wantFileOffset int
	}{
		// Page 1: files 0-49
		{page: 1, wantFolderLimit: 0, wantFolderOffset: 0, wantFileLimit: 50, wantFileOffset: 0},
		// Page 2: files 50-99
		{page: 2, wantFolderLimit: 0, wantFolderOffset: 0, wantFileLimit: 50, wantFileOffset: 50},
		// Page 3: files 100-149
		{page: 3, wantFolderLimit: 0, wantFolderOffset: 0, wantFileLimit: 50, wantFileOffset: 100},
		// Page 4: files 150-199
		{page: 4, wantFolderLimit: 0, wantFolderOffset: 0, wantFileLimit: 50, wantFileOffset: 150},
	}

	for _, tt := range tests {
		t.Run("page_"+string(rune(tt.page+'0')), func(t *testing.T) {
			folderLimit, folderOffset, fileLimit, fileOffset := CalculateFolderFileOffsets(
				tt.page, pageSize, totalFolders, totalFiles,
			)

			if folderLimit != tt.wantFolderLimit {
				t.Errorf("folderLimit = %d, want %d", folderLimit, tt.wantFolderLimit)
			}
			if folderOffset != tt.wantFolderOffset {
				t.Errorf("folderOffset = %d, want %d", folderOffset, tt.wantFolderOffset)
			}
			if fileLimit != tt.wantFileLimit {
				t.Errorf("fileLimit = %d, want %d", fileLimit, tt.wantFileLimit)
			}
			if fileOffset != tt.wantFileOffset {
				t.Errorf("fileOffset = %d, want %d", fileOffset, tt.wantFileOffset)
			}
		})
	}
}

// TestCalculateFolderFileOffsets_Transition tests the critical transition from folders to files
func TestCalculateFolderFileOffsets_Transition(t *testing.T) {
	const pageSize = 50
	totalFolders := int64(30)
	totalFiles := int64(200)

	tests := []struct {
		name                                                           string
		page                                                           int
		wantFolderLimit, wantFolderOffset, wantFileLimit, wantFileOffset int
	}{
		{
			name:            "Page 1: 30 folders + 20 files",
			page:            1,
			wantFolderLimit: 30, wantFolderOffset: 0,
			wantFileLimit: 20, wantFileOffset: 0,
		},
		{
			name:            "Page 2: 50 files (files 20-69)",
			page:            2,
			wantFolderLimit: 0, wantFolderOffset: 0,
			wantFileLimit: 50, wantFileOffset: 20,
		},
		{
			name:            "Page 3: 50 files (files 70-119)",
			page:            3,
			wantFolderLimit: 0, wantFolderOffset: 0,
			wantFileLimit: 50, wantFileOffset: 70,
		},
		{
			name:            "Page 4: 50 files (files 120-169)",
			page:            4,
			wantFolderLimit: 0, wantFolderOffset: 0,
			wantFileLimit: 50, wantFileOffset: 120,
		},
		{
			name:            "Page 5: 30 files (files 170-199, partial page)",
			page:            5,
			wantFolderLimit: 0, wantFolderOffset: 0,
			wantFileLimit: 50, wantFileOffset: 170,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			folderLimit, folderOffset, fileLimit, fileOffset := CalculateFolderFileOffsets(
				tt.page, pageSize, totalFolders, totalFiles,
			)

			if folderLimit != tt.wantFolderLimit {
				t.Errorf("folderLimit = %d, want %d", folderLimit, tt.wantFolderLimit)
			}
			if folderOffset != tt.wantFolderOffset {
				t.Errorf("folderOffset = %d, want %d", folderOffset, tt.wantFolderOffset)
			}
			if fileLimit != tt.wantFileLimit {
				t.Errorf("fileLimit = %d, want %d", fileLimit, tt.wantFileLimit)
			}
			if fileOffset != tt.wantFileOffset {
				t.Errorf("fileOffset = %d, want %d", fileOffset, tt.wantFileOffset)
			}
		})
	}
}

// TestCalculateFolderFileOffsets_EdgeCases tests edge cases and boundary conditions
func TestCalculateFolderFileOffsets_EdgeCases(t *testing.T) {
	tests := []struct {
		name                                                           string
		page, pageSize                                                 int
		totalFolders, totalFiles                                       int64
		wantFolderLimit, wantFolderOffset, wantFileLimit, wantFileOffset int
	}{
		{
			name:         "Empty - no folders, no files",
			page:         1,
			pageSize:     50,
			totalFolders: 0,
			totalFiles:   0,
			wantFolderLimit: 0, wantFolderOffset: 0,
			wantFileLimit: 0, wantFileOffset: 0,
		},
		{
			name:         "Exactly one page - 50 folders, 0 files",
			page:         1,
			pageSize:     50,
			totalFolders: 50,
			totalFiles:   0,
			wantFolderLimit: 50, wantFolderOffset: 0,
			wantFileLimit: 0, wantFileOffset: 0,
		},
		{
			name:         "Exactly one page - 0 folders, 50 files",
			page:         1,
			pageSize:     50,
			totalFolders: 0,
			totalFiles:   50,
			wantFolderLimit: 0, wantFolderOffset: 0,
			wantFileLimit: 50, wantFileOffset: 0,
		},
		{
			name:         "Exactly one page - 25 folders, 25 files",
			page:         1,
			pageSize:     50,
			totalFolders: 25,
			totalFiles:   25,
			wantFolderLimit: 25, wantFolderOffset: 0,
			wantFileLimit: 25, wantFileOffset: 0,
		},
		{
			name:         "Partial first page - 10 folders, 10 files",
			page:         1,
			pageSize:     50,
			totalFolders: 10,
			totalFiles:   10,
			wantFolderLimit: 10, wantFolderOffset: 0,
			wantFileLimit: 40, wantFileOffset: 0,
		},
		{
			name:         "Single folder",
			page:         1,
			pageSize:     50,
			totalFolders: 1,
			totalFiles:   0,
			wantFolderLimit: 1, wantFolderOffset: 0,
			wantFileLimit: 0, wantFileOffset: 0,
		},
		{
			name:         "Single file",
			page:         1,
			pageSize:     50,
			totalFolders: 0,
			totalFiles:   1,
			wantFolderLimit: 0, wantFolderOffset: 0,
			wantFileLimit: 50, wantFileOffset: 0,
		},
		{
			name:         "Large numbers - 10000 folders, page 100",
			page:         100,
			pageSize:     50,
			totalFolders: 10000,
			totalFiles:   5000,
			wantFolderLimit: 50, wantFolderOffset: 4950,
			wantFileLimit: 0, wantFileOffset: 0,
		},
		{
			name:         "Transition at exact boundary - 50 folders, page 2",
			page:         2,
			pageSize:     50,
			totalFolders: 50,
			totalFiles:   100,
			wantFolderLimit: 0, wantFolderOffset: 0,
			wantFileLimit: 50, wantFileOffset: 0,
		},
		{
			name:         "Transition split - 51 folders means page 2 has 1 folder + 49 files",
			page:         2,
			pageSize:     50,
			totalFolders: 51,
			totalFiles:   100,
			wantFolderLimit: 1, wantFolderOffset: 50,
			wantFileLimit: 49, wantFileOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			folderLimit, folderOffset, fileLimit, fileOffset := CalculateFolderFileOffsets(
				tt.page, tt.pageSize, tt.totalFolders, tt.totalFiles,
			)

			if folderLimit != tt.wantFolderLimit {
				t.Errorf("folderLimit = %d, want %d", folderLimit, tt.wantFolderLimit)
			}
			if folderOffset != tt.wantFolderOffset {
				t.Errorf("folderOffset = %d, want %d", folderOffset, tt.wantFolderOffset)
			}
			if fileLimit != tt.wantFileLimit {
				t.Errorf("fileLimit = %d, want %d", fileLimit, tt.wantFileLimit)
			}
			if fileOffset != tt.wantFileOffset {
				t.Errorf("fileOffset = %d, want %d", fileOffset, tt.wantFileOffset)
			}
		})
	}
}

// TestCalculateFolderFileOffsets_LastPagePartial tests that the last page handles partial results correctly
func TestCalculateFolderFileOffsets_LastPagePartial(t *testing.T) {
	// 30 folders + 175 files = 205 total items
	// With pageSize=50: pages 1-4 are full, page 5 is partial (5 items)
	const pageSize = 50
	totalFolders := int64(30)
	totalFiles := int64(175)

	// Page 5: should request 50 files starting at offset 170, but only 5 actually exist
	// The function calculates what to REQUEST, not what will be RETURNED
	folderLimit, folderOffset, fileLimit, fileOffset := CalculateFolderFileOffsets(
		5, pageSize, totalFolders, totalFiles,
	)

	if folderLimit != 0 {
		t.Errorf("folderLimit = %d, want 0", folderLimit)
	}
	if folderOffset != 0 {
		t.Errorf("folderOffset = %d, want 0", folderOffset)
	}
	if fileLimit != 50 {
		t.Errorf("fileLimit = %d, want 50", fileLimit)
	}
	if fileOffset != 170 {
		t.Errorf("fileOffset = %d, want 170", fileOffset)
	}
}

// TestCalculateFolderFileOffsets_DifferentPageSizes tests with various page sizes
func TestCalculateFolderFileOffsets_DifferentPageSizes(t *testing.T) {
	tests := []struct {
		name                                                           string
		page, pageSize                                                 int
		totalFolders, totalFiles                                       int64
		wantFolderLimit, wantFolderOffset, wantFileLimit, wantFileOffset int
	}{
		{
			name:         "PageSize 10 - page 1",
			page:         1,
			pageSize:     10,
			totalFolders: 5,
			totalFiles:   20,
			wantFolderLimit: 5, wantFolderOffset: 0,
			wantFileLimit: 5, wantFileOffset: 0,
		},
		{
			name:         "PageSize 100 - page 1",
			page:         1,
			pageSize:     100,
			totalFolders: 30,
			totalFiles:   200,
			wantFolderLimit: 30, wantFolderOffset: 0,
			wantFileLimit: 70, wantFileOffset: 0,
		},
		{
			name:         "PageSize 1 - page 1 (only first folder)",
			page:         1,
			pageSize:     1,
			totalFolders: 10,
			totalFiles:   10,
			wantFolderLimit: 1, wantFolderOffset: 0,
			wantFileLimit: 0, wantFileOffset: 0,
		},
		{
			name:         "PageSize 1 - page 11 (first file)",
			page:         11,
			pageSize:     1,
			totalFolders: 10,
			totalFiles:   10,
			wantFolderLimit: 0, wantFolderOffset: 0,
			wantFileLimit: 1, wantFileOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			folderLimit, folderOffset, fileLimit, fileOffset := CalculateFolderFileOffsets(
				tt.page, tt.pageSize, tt.totalFolders, tt.totalFiles,
			)

			if folderLimit != tt.wantFolderLimit {
				t.Errorf("folderLimit = %d, want %d", folderLimit, tt.wantFolderLimit)
			}
			if folderOffset != tt.wantFolderOffset {
				t.Errorf("folderOffset = %d, want %d", folderOffset, tt.wantFolderOffset)
			}
			if fileLimit != tt.wantFileLimit {
				t.Errorf("fileLimit = %d, want %d", fileLimit, tt.wantFileLimit)
			}
			if fileOffset != tt.wantFileOffset {
				t.Errorf("fileOffset = %d, want %d", fileOffset, tt.wantFileOffset)
			}
		})
	}
}
