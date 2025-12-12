package dbsvc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCalculateFolderFileOffsets_VerifyNoOffByOneErrors tests comprehensive scenarios
// to ensure no off-by-one errors exist in the pagination offset calculations
func TestCalculateFolderFileOffsets_VerifyNoOffByOneErrors(t *testing.T) {
	tests := []struct {
		name                                                           string
		page, pageSize                                                 int
		totalFolders, totalFiles                                       int64
		wantFolderLimit, wantFolderOffset, wantFileLimit, wantFileOffset int
	}{
		{
			name:         "Boundary: 49 folders + 1 file (page 1)",
			page:         1,
			pageSize:     50,
			totalFolders: 49,
			totalFiles:   1,
			wantFolderLimit: 49, wantFolderOffset: 0,
			wantFileLimit: 1, wantFileOffset: 0,
		},
		{
			name:         "Boundary: 51 folders (page 2 has 1 folder only)",
			page:         2,
			pageSize:     50,
			totalFolders: 51,
			totalFiles:   0,
			wantFolderLimit: 1, wantFolderOffset: 50,
			wantFileLimit: 0, wantFileOffset: 0,
		},
		{
			name:         "Boundary: 100 folders exactly (page 2)",
			page:         2,
			pageSize:     50,
			totalFolders: 100,
			totalFiles:   50,
			wantFolderLimit: 50, wantFolderOffset: 50,
			wantFileLimit: 0, wantFileOffset: 0,
		},
		{
			name:         "Boundary: 100 folders exactly (page 3 starts files)",
			page:         3,
			pageSize:     50,
			totalFolders: 100,
			totalFiles:   50,
			wantFolderLimit: 0, wantFolderOffset: 0,
			wantFileLimit: 50, wantFileOffset: 0,
		},
		{
			name:         "Large dataset: 1000 folders, 5000 files, page 50",
			page:         50,
			pageSize:     50,
			totalFolders: 1000,
			totalFiles:   5000,
			wantFolderLimit: 0, wantFolderOffset: 0,
			wantFileLimit: 50, wantFileOffset: 1450,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			folderLimit, folderOffset, fileLimit, fileOffset := CalculateFolderFileOffsets(
				tt.page, tt.pageSize, tt.totalFolders, tt.totalFiles,
			)

			assert.Equal(t, tt.wantFolderLimit, folderLimit, "folderLimit mismatch")
			assert.Equal(t, tt.wantFolderOffset, folderOffset, "folderOffset mismatch")
			assert.Equal(t, tt.wantFileLimit, fileLimit, "fileLimit mismatch")
			assert.Equal(t, tt.wantFileOffset, fileOffset, "fileOffset mismatch")
		})
	}
}

// TestGetDirectChildrenPaginated_VerifyMethodSignature ensures the method exists with correct signature
func TestGetDirectChildrenPaginated_VerifyMethodSignature(t *testing.T) {
	// This test verifies the method signature compiles correctly
	var s *Service
	if s != nil {
		ctx := context.Background()
		folders, files, totalFolders, totalFiles, err := s.GetDirectChildrenPaginated(
			ctx, "test-bucket", "test-prefix/", 1, 50,
		)

		// Type assertions to verify return types
		_ = folders  // []dto.S3Object
		_ = files    // []dto.S3Object
		_ = totalFolders // int64
		_ = totalFiles   // int64
		_ = err      // error
	}
}

// TestCountDirectChildren_VerifyMethodSignature ensures the method exists with correct signature
func TestCountDirectChildren_VerifyMethodSignature(t *testing.T) {
	// This test verifies the method signature compiles correctly
	var s *Service
	if s != nil {
		ctx := context.Background()
		totalFolders, totalFiles, err := s.CountDirectChildren(
			ctx, "test-bucket", "test-prefix/",
		)

		// Type assertions to verify return types
		_ = totalFolders // int64
		_ = totalFiles   // int64
		_ = err          // error
	}
}

// Note: Full integration tests for GetDirectChildrenPaginated and CountDirectChildren
// would require:
// 1. Test database setup (PostgreSQL container via testcontainers-go)
// 2. Running migrations
// 3. Seeding test data with known folder/file structures
// 4. Testing pagination across multiple pages
// 5. Verifying folder-first ordering is maintained
// 6. Testing edge cases (empty folders, exact page boundaries, etc.)
//
// Example test structure for future integration tests:
//
// func TestGetDirectChildrenPaginated_Integration(t *testing.T) {
//     if testing.Short() {
//         t.Skip("Skipping integration test")
//     }
//
//     // Setup: Start PostgreSQL container
//     ctx := context.Background()
//     container, connString := setupTestDatabase(t, ctx)
//     defer container.Terminate(ctx)
//
//     // Run migrations
//     db, err := sql.Open("postgres", connString)
//     require.NoError(t, err)
//     defer db.Close()
//     runMigrations(t, db)
//
//     // Create service
//     queries := database.New(db)
//     service := &Service{queries: queries}
//
//     // Seed test data: 100 folders + 500 files in "test-prefix/"
//     seedTestData(t, queries, "test-bucket", "test-prefix/", 100, 500)
//
//     // Test page 1: Should have 50 folders
//     folders, files, totalFolders, totalFiles, err := service.GetDirectChildrenPaginated(
//         ctx, "test-bucket", "test-prefix/", 1, 50,
//     )
//     require.NoError(t, err)
//     assert.Equal(t, int64(100), totalFolders)
//     assert.Equal(t, int64(500), totalFiles)
//     assert.Len(t, folders, 50)
//     assert.Len(t, files, 0)
//
//     // Test page 2: Should have 50 folders
//     folders, files, _, _, err = service.GetDirectChildrenPaginated(
//         ctx, "test-bucket", "test-prefix/", 2, 50,
//     )
//     require.NoError(t, err)
//     assert.Len(t, folders, 50)
//     assert.Len(t, files, 0)
//
//     // Test page 3: Should have 50 files (all folders consumed)
//     folders, files, _, _, err = service.GetDirectChildrenPaginated(
//         ctx, "test-bucket", "test-prefix/", 3, 50,
//     )
//     require.NoError(t, err)
//     assert.Len(t, folders, 0)
//     assert.Len(t, files, 50)
//
//     // Verify no duplicates across pages
//     allKeys := make(map[string]bool)
//     for i := 1; i <= 12; i++ {
//         folders, files, _, _, _ := service.GetDirectChildrenPaginated(
//             ctx, "test-bucket", "test-prefix/", i, 50,
//         )
//         for _, obj := range append(folders, files...) {
//             assert.False(t, allKeys[obj.Key], "Duplicate key found: %s", obj.Key)
//             allKeys[obj.Key] = true
//         }
//     }
//     assert.Equal(t, 600, len(allKeys), "Should have exactly 600 unique items")
// }
