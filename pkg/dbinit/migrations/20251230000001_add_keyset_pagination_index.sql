-- migrate:up
-- Add composite index for efficient keyset pagination
-- This index supports queries with row value comparison: WHERE (is_folder, key) < (?, ?)
-- Index column order optimized for: bucket_id filter, prefix filter, then (is_folder, key) ordering
-- Note: CONCURRENTLY removed to allow running inside migration transaction
-- For production with large tables, consider creating this index manually with CONCURRENTLY
CREATE INDEX IF NOT EXISTS idx_s3_objects_keyset
ON s3_objects (bucket_id, prefix, is_folder DESC, key ASC);

-- migrate:down
-- Remove keyset pagination index
DROP INDEX IF EXISTS idx_s3_objects_keyset;
