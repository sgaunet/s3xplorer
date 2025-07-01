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

-- name: UpdateScanJobStatus :one
UPDATE scan_jobs
SET status = $2,
    started_at = CASE WHEN $2 = 'running' THEN NOW() ELSE started_at END,
    completed_at = CASE WHEN $2 IN ('completed', 'failed') THEN NOW() ELSE completed_at END,
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