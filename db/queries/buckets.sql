-- name: GetBucket :one
SELECT * FROM buckets
WHERE name = $1;

-- name: GetBucketByID :one
SELECT * FROM buckets
WHERE id = $1;

-- name: ListBuckets :many
SELECT * FROM buckets
ORDER BY name;

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