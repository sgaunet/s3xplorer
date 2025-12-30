-- name: GetS3Object :one
SELECT * FROM s3_objects
WHERE bucket_id = $1 AND key = $2;

-- name: GetCursorForListS3Objects :one
-- Get cursor position for keyset pagination
SELECT is_folder, key FROM s3_objects
WHERE bucket_id = $1
  AND ($2 = '' OR prefix = $2)
ORDER BY is_folder DESC, key ASC
LIMIT 1 OFFSET $3;

-- name: ListS3Objects :many
SELECT * FROM s3_objects
WHERE bucket_id = $1
  AND ($2 = '' OR prefix = $2)
  AND (sqlc.narg('cursor_is_folder')::boolean IS NULL
       OR (is_folder, key) < (sqlc.narg('cursor_is_folder')::boolean, sqlc.narg('cursor_key')::text))
ORDER BY is_folder DESC, key ASC
LIMIT $3;

-- name: GetCursorForListS3ObjectsByPrefix :one
-- Get cursor position for keyset pagination with prefix search
SELECT is_folder, key FROM s3_objects
WHERE bucket_id = $1
  AND prefix LIKE $2 || '%'
ORDER BY is_folder DESC, key ASC
LIMIT 1 OFFSET $3;

-- name: ListS3ObjectsByPrefix :many
SELECT * FROM s3_objects
WHERE bucket_id = $1
  AND prefix LIKE $2 || '%'
  AND (sqlc.narg('cursor_is_folder')::boolean IS NULL
       OR (is_folder, key) < (sqlc.narg('cursor_is_folder')::boolean, sqlc.narg('cursor_key')::text))
ORDER BY is_folder DESC, key ASC
LIMIT $3;

-- name: GetCursorForSearchS3Objects :one
-- Get cursor position for keyset pagination in search results
SELECT is_folder, key FROM s3_objects
WHERE bucket_id = $1
  AND key ILIKE '%' || $2 || '%'
ORDER BY is_folder DESC, key ASC
LIMIT 1 OFFSET $3;

-- name: SearchS3Objects :many
SELECT * FROM s3_objects
WHERE bucket_id = $1
  AND key ILIKE '%' || $2 || '%'
  AND (sqlc.narg('cursor_is_folder')::boolean IS NULL
       OR (is_folder, key) < (sqlc.narg('cursor_is_folder')::boolean, sqlc.narg('cursor_key')::text))
ORDER BY is_folder DESC, key ASC
LIMIT $3;

-- name: CountS3Objects :one
SELECT COUNT(*) FROM s3_objects
WHERE bucket_id = $1 
  AND ($2 = '' OR prefix = $2);

-- name: GetCursorForListS3Folders :one
-- Get cursor position for keyset pagination of folders only
SELECT key FROM s3_objects
WHERE bucket_id = $1
  AND (
    -- Handle root level (empty prefix): objects with empty or null prefix
    ($2 = '' AND (prefix = '' OR prefix IS NULL))
    OR
    -- Handle non-empty prefix: exact prefix match
    ($2 != '' AND prefix = $2)
  )
  AND is_folder = true
ORDER BY key ASC
LIMIT 1 OFFSET $3;

-- name: ListS3Folders :many
SELECT * FROM s3_objects
WHERE bucket_id = $1
  AND (
    -- Handle root level (empty prefix): objects with empty or null prefix
    ($2 = '' AND (prefix = '' OR prefix IS NULL))
    OR
    -- Handle non-empty prefix: exact prefix match
    ($2 != '' AND prefix = $2)
  )
  AND is_folder = true
  AND (sqlc.narg('cursor_key')::text IS NULL OR key > sqlc.narg('cursor_key'))
ORDER BY key ASC
LIMIT $3;

-- name: GetCursorForListS3Files :one
-- Get cursor position for keyset pagination of files only
SELECT key FROM s3_objects
WHERE bucket_id = $1
  AND (
    -- Handle root level (empty prefix): objects with empty or null prefix
    ($2 = '' AND (prefix = '' OR prefix IS NULL))
    OR
    -- Handle non-empty prefix: exact prefix match
    ($2 != '' AND prefix = $2)
  )
  AND is_folder = false
ORDER BY key ASC
LIMIT 1 OFFSET $3;

-- name: ListS3Files :many
SELECT * FROM s3_objects
WHERE bucket_id = $1
  AND (
    -- Handle root level (empty prefix): objects with empty or null prefix
    ($2 = '' AND (prefix = '' OR prefix IS NULL))
    OR
    -- Handle non-empty prefix: exact prefix match
    ($2 != '' AND prefix = $2)
  )
  AND is_folder = false
  AND (sqlc.narg('cursor_key')::text IS NULL OR key > sqlc.narg('cursor_key'))
ORDER BY key ASC
LIMIT $3;

-- name: CreateS3Object :one
INSERT INTO s3_objects (bucket_id, key, size, last_modified, etag, storage_class, is_folder, prefix)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (bucket_id, key) DO UPDATE SET
    size = EXCLUDED.size,
    last_modified = EXCLUDED.last_modified,
    etag = EXCLUDED.etag,
    storage_class = EXCLUDED.storage_class,
    is_folder = EXCLUDED.is_folder,
    prefix = EXCLUDED.prefix,
    marked_for_deletion = FALSE,
    updated_at = NOW()
RETURNING *;

-- name: UpdateS3Object :one
UPDATE s3_objects
SET size = $3,
    last_modified = $4,
    etag = $5,
    storage_class = $6,
    updated_at = NOW()
WHERE bucket_id = $1 AND key = $2
RETURNING *;

-- name: DeleteS3Object :exec
DELETE FROM s3_objects
WHERE bucket_id = $1 AND key = $2;

-- name: DeleteS3ObjectsByBucket :exec
DELETE FROM s3_objects
WHERE bucket_id = $1;

-- name: GetCursorForDirectChildren :one
-- Get cursor position for keyset pagination of direct children
SELECT is_folder, key FROM s3_objects
WHERE bucket_id = $1
  AND (
    -- Handle root level (empty prefix): objects with empty or null prefix
    ($2 = '' AND (prefix = '' OR prefix IS NULL))
    OR
    -- Handle non-empty prefix: exact prefix match
    ($2 != '' AND prefix = $2)
  )
  AND key != $2
  AND (
    -- Direct files: files whose prefix exactly matches the given prefix
    (is_folder = false)
    OR
    -- Direct folders: folders whose prefix exactly matches the given prefix
    (is_folder = true)
  )
ORDER BY is_folder DESC, key ASC
LIMIT 1 OFFSET $3;

-- name: GetDirectChildren :many
-- Get only immediate children (files and folders) under a specific prefix
-- For hierarchical navigation - not recursive
SELECT * FROM s3_objects
WHERE bucket_id = $1
  AND (
    -- Handle root level (empty prefix): objects with empty or null prefix
    ($2 = '' AND (prefix = '' OR prefix IS NULL))
    OR
    -- Handle non-empty prefix: exact prefix match
    ($2 != '' AND prefix = $2)
  )
  AND key != $2
  AND (
    -- Direct files: files whose prefix exactly matches the given prefix
    (is_folder = false)
    OR
    -- Direct folders: folders whose prefix exactly matches the given prefix
    (is_folder = true)
  )
  AND (sqlc.narg('cursor_is_folder')::boolean IS NULL
       OR (is_folder, key) < (sqlc.narg('cursor_is_folder')::boolean, sqlc.narg('cursor_key')::text))
ORDER BY is_folder DESC, key ASC
LIMIT $3;

-- name: GetParentFolder :one
-- Get parent folder information for breadcrumb navigation
SELECT * FROM s3_objects
WHERE bucket_id = $1 
  AND key = $2
  AND is_folder = true
LIMIT 1;

-- name: GetBreadcrumbPath :many
-- Get all parent folders for breadcrumb navigation
SELECT * FROM s3_objects
WHERE bucket_id = $1 
  AND is_folder = true
  AND $2 LIKE key || '%'
  AND key != $2
ORDER BY LENGTH(key) ASC;

-- name: MarkAllObjectsForDeletion :exec
-- Mark all objects in a bucket as potentially deleted before scanning
UPDATE s3_objects
SET marked_for_deletion = TRUE,
    updated_at = NOW()
WHERE bucket_id = $1;

-- name: UnmarkObjectForDeletion :exec
-- Unmark a specific object as not deleted (found during scan)
UPDATE s3_objects
SET marked_for_deletion = FALSE,
    updated_at = NOW()
WHERE bucket_id = $1 AND key = $2;

-- name: DeleteMarkedObjects :exec
-- Delete all objects that are still marked for deletion after scan
DELETE FROM s3_objects
WHERE bucket_id = $1 AND marked_for_deletion = TRUE;

-- name: CountMarkedObjects :one
-- Count objects marked for deletion
SELECT COUNT(*) FROM s3_objects
WHERE bucket_id = $1 AND marked_for_deletion = TRUE;

-- name: CountDirectChildrenFolders :one
-- Count only immediate child folders under a specific prefix
-- For pagination - not recursive
SELECT COUNT(*) FROM s3_objects
WHERE bucket_id = $1
  AND (
    -- Handle root level (empty prefix): objects with empty or null prefix
    ($2 = '' AND (prefix = '' OR prefix IS NULL))
    OR
    -- Handle non-empty prefix: exact prefix match
    ($2 != '' AND prefix = $2)
  )
  AND key != $2
  AND is_folder = true;

-- name: CountDirectChildrenFiles :one
-- Count only immediate child files under a specific prefix
-- For pagination - not recursive
SELECT COUNT(*) FROM s3_objects
WHERE bucket_id = $1
  AND (
    -- Handle root level (empty prefix): objects with empty or null prefix
    ($2 = '' AND (prefix = '' OR prefix IS NULL))
    OR
    -- Handle non-empty prefix: exact prefix match
    ($2 != '' AND prefix = $2)
  )
  AND key != $2
  AND is_folder = false;