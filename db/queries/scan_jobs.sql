-- name: GetScanJob :one
SELECT * FROM scan_jobs
WHERE id = $1;

-- name: ListScanJobs :many
SELECT * FROM scan_jobs
WHERE bucket_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetLatestScanJob :one
SELECT * FROM scan_jobs
WHERE bucket_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: CreateScanJob :one
INSERT INTO scan_jobs (bucket_id, status)
VALUES ($1, $2)
RETURNING *;

-- name: CreateGlobalScanJob :one
INSERT INTO scan_jobs (bucket_id, status)
VALUES (NULL, $1)
RETURNING *;

-- name: UpdateScanJobStatus :one
UPDATE scan_jobs
SET status = $2::text,
    started_at = CASE 
        WHEN $2::text = 'running' THEN NOW() 
        ELSE started_at 
    END,
    completed_at = CASE 
        WHEN $2::text = 'completed' THEN NOW()
        WHEN $2::text = 'failed' THEN NOW()
        ELSE completed_at 
    END,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateScanJobProgress :one
UPDATE scan_jobs
SET objects_scanned = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateScanJobError :one
UPDATE scan_jobs
SET status = 'failed',
    error_message = $2,
    completed_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateScanJobStats :one
UPDATE scan_jobs
SET objects_scanned = $2,
    objects_created = $3,
    objects_updated = $4,
    objects_deleted = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateScanJobBucketStats :one
UPDATE scan_jobs
SET buckets_validated = $2,
    buckets_marked_inaccessible = $3,
    buckets_cleaned_up = $4,
    bucket_validation_errors = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateScanJobFullStats :one
UPDATE scan_jobs
SET objects_scanned = $2,
    objects_created = $3,
    objects_updated = $4,
    objects_deleted = $5,
    buckets_validated = $6,
    buckets_marked_inaccessible = $7,
    buckets_cleaned_up = $8,
    bucket_validation_errors = $9,
    updated_at = NOW()
WHERE id = $1
RETURNING *;