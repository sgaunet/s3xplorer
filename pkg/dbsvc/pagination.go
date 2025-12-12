package dbsvc

// CalculateFolderFileOffsets calculates the database offsets and limits for folder-first pagination.
// This function implements the logic for displaying folders before files in paginated results.
//
// The pagination strategy is:
//  1. Folders are displayed first (sorted alphabetically)
//  2. Files are displayed after all folders (sorted alphabetically)
//  3. Each page shows up to pageSize items
//  4. A page may contain only folders, only files, or a mix of both
//
// Parameters:
//   - page: Current page number (1-indexed)
//   - pageSize: Number of items per page
//   - totalFolders: Total number of folders available
//   - totalFiles: Total number of files available
//
// Returns:
//   - folderLimit: Number of folders to fetch (0 if no folders needed)
//   - folderOffset: Starting position for folder query (0-indexed)
//   - fileLimit: Number of files to fetch (0 if no files needed)
//   - fileOffset: Starting position for file query (0-indexed)
//
// Example scenarios:
//   - 30 folders, 200 files, pageSize=50, page=1: Returns 30 folders (0-29) + 20 files (0-19)
//   - 30 folders, 200 files, pageSize=50, page=2: Returns 50 files (20-69)
//   - 120 folders, 0 files, pageSize=50, page=2: Returns 50 folders (50-99)
//
//nolint:nonamedreturns // Named returns improve readability for 4 int return values
func CalculateFolderFileOffsets(
	page, pageSize int,
	totalFolders, totalFiles int64,
) (folderLimit, folderOffset, fileLimit, fileOffset int) {
	// Calculate 0-indexed position range for this page
	startIdx := (page - 1) * pageSize // First item position (0-indexed)
	endIdx := startIdx + pageSize     // Last item position + 1 (exclusive, 0-indexed)

	// Initialize all return values to 0
	folderLimit, folderOffset, fileLimit, fileOffset = 0, 0, 0, 0

	// Determine if we need to fetch folders
	if startIdx < int(totalFolders) {
		// This page starts within the folder range
		folderOffset = startIdx
		// Calculate how many folders we can fit on this page
		// It's either pageSize or the remaining folders, whichever is smaller
		folderLimit = min(pageSize, int(totalFolders)-startIdx)
	}

	// Determine if we need to fetch files
	if endIdx > int(totalFolders) && totalFiles > 0 {
		// This page extends past the folders into the files range
		// Calculate the starting position for files (0-indexed within the file collection)
		fileOffset = max(0, startIdx-int(totalFolders))
		// Calculate how many files we need to fill the rest of the page
		fileLimit = pageSize - folderLimit
	}

	return folderLimit, folderOffset, fileLimit, fileOffset
}
