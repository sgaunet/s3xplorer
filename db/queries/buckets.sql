-- name: GetBucket :one
SELECT * FROM buckets
WHERE name = $1;

-- name: GetBucketByID :one
SELECT * FROM buckets
WHERE id = $1;

-- name: ListBuckets :many
SELECT * FROM buckets
ORDER BY name;

-- name: ListAccessibleBuckets :many
-- Returns only buckets that are accessible (not marked for deletion and no recent permanent failures)
SELECT b.* FROM buckets b
WHERE b.marked_for_deletion = false
  AND (b.access_error IS NULL OR b.last_accessible_at > NOW() - INTERVAL '24 hours')
ORDER BY b.name;

-- name: ListBucketsWithStatus :many
-- Returns all buckets with their latest scan job status for admin/debug purposes
SELECT 
  b.*,
  sj.status as latest_scan_status,
  sj.error_message as latest_scan_error,
  sj.completed_at as latest_scan_completed_at
FROM buckets b
LEFT JOIN LATERAL (
  SELECT status, error_message, completed_at
  FROM scan_jobs 
  WHERE bucket_id = b.id 
  ORDER BY created_at DESC 
  LIMIT 1
) sj ON true
ORDER BY b.name;

-- name: CreateBucket :one
INSERT INTO buckets (name, region)
VALUES ($1, $2)
ON CONFLICT (name) DO UPDATE SET
    region = EXCLUDED.region,
    updated_at = NOW()
RETURNING *;

-- name: DeleteBucket :exec
DELETE FROM buckets
WHERE id = $1;

-- Bucket lifecycle management queries for bucket synchronization

-- name: MarkAllBucketsForDeletion :exec
UPDATE buckets
SET marked_for_deletion = true
WHERE marked_for_deletion = false;

-- name: MarkBucketForDeletion :exec
UPDATE buckets
SET marked_for_deletion = true
WHERE id = $1;

-- name: UnmarkBucketForDeletion :exec
UPDATE buckets
SET marked_for_deletion = false,
    last_accessible_at = NOW(),
    access_error = NULL
WHERE id = $1;

-- name: UpdateBucketAccessError :exec
UPDATE buckets
SET access_error = $2,
    last_accessible_at = NOW()
WHERE id = $1;

-- name: GetBucketsMarkedForDeletion :many
SELECT * FROM buckets
WHERE marked_for_deletion = true;

-- name: GetInaccessibleBuckets :many
SELECT * FROM buckets
WHERE last_accessible_at < NOW() - INTERVAL '1 hour' * $1
   OR (last_accessible_at IS NULL AND created_at < NOW() - INTERVAL '1 hour' * $1);

-- name: GetBucketsToDelete :many
SELECT * FROM buckets
WHERE marked_for_deletion = true
  AND (last_accessible_at < NOW() - INTERVAL '1 hour' * $1
       OR (last_accessible_at IS NULL AND created_at < NOW() - INTERVAL '1 hour' * $1));

-- name: DeleteMarkedBuckets :exec
DELETE FROM buckets
WHERE marked_for_deletion = true
  AND (last_accessible_at < NOW() - INTERVAL '1 hour' * $1
       OR (last_accessible_at IS NULL AND created_at < NOW() - INTERVAL '1 hour' * $1));

-- name: CountMarkedBuckets :one
SELECT COUNT(*) FROM buckets
WHERE marked_for_deletion = true;