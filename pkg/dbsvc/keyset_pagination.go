package dbsvc

import (
	"context"
	"database/sql"
	"errors"

	"github.com/sgaunet/s3xplorer/pkg/database"
)

// KeysetCursor represents a pagination cursor for keyset pagination.
// It contains the last item's sort keys to enable efficient "seek" queries.
type KeysetCursor struct {
	IsFolder bool
	Key      string
}

// GetCursorForPage retrieves the cursor for a given page number.
// Returns nil cursor for page 1 (start from beginning).
// Returns nil cursor if offset is beyond total items (graceful degradation).
func (s *Service) GetCursorForPage(
	ctx context.Context,
	bucketID int64,
	prefix string,
	page int,
	pageSize int,
) (*KeysetCursor, error) {
	if page <= 1 {
		return nil, nil //nolint:nilnil // Returning nil cursor is intentional for page 1
	}

	offset := int64((page - 1) * pageSize)

	// Get cursor at offset position (last item of previous page)
	cursor, err := s.queries.GetCursorForDirectChildren(ctx, database.GetCursorForDirectChildrenParams{
		BucketID: int32(bucketID),
		Column2:  prefix,
		Offset:   int32(offset - 1), // Get the last item of previous page
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Offset beyond total items - return nil to indicate no more results
			return nil, nil //nolint:nilnil // Returning nil cursor is intentional for graceful degradation
		}
		return nil, err
	}

	return &KeysetCursor{
		IsFolder: cursor.IsFolder.Bool,
		Key:      cursor.Key,
	}, nil
}

// GetCursorForFolders retrieves the cursor for folder-only pagination.
func (s *Service) GetCursorForFolders(
	ctx context.Context,
	bucketID int64,
	prefix string,
	page int,
	pageSize int,
) (*string, error) {
	if page <= 1 {
		return nil, nil //nolint:nilnil // Returning nil cursor is intentional for page 1
	}

	offset := int64((page - 1) * pageSize)

	cursorKey, err := s.queries.GetCursorForListS3Folders(ctx, database.GetCursorForListS3FoldersParams{
		BucketID: int32(bucketID),
		Column2:  prefix,
		Offset:   int32(offset - 1),
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // Returning nil cursor is intentional for graceful degradation
		}
		return nil, err
	}

	return &cursorKey, nil
}

// GetCursorForFiles retrieves the cursor for file-only pagination.
func (s *Service) GetCursorForFiles(
	ctx context.Context,
	bucketID int64,
	prefix string,
	page int,
	pageSize int,
) (*string, error) {
	if page <= 1 {
		return nil, nil //nolint:nilnil // Returning nil cursor is intentional for page 1
	}

	offset := int64((page - 1) * pageSize)

	cursorKey, err := s.queries.GetCursorForListS3Files(ctx, database.GetCursorForListS3FilesParams{
		BucketID: int32(bucketID),
		Column2:  prefix,
		Offset:   int32(offset - 1),
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil //nolint:nilnil // Returning nil cursor is intentional for graceful degradation
		}
		return nil, err
	}

	return &cursorKey, nil
}
