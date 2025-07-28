-- migrate:up
-- Allow global scan jobs by making bucket_id nullable
-- Global scan jobs (multi-bucket scans) will have bucket_id = NULL
ALTER TABLE scan_jobs ALTER COLUMN bucket_id DROP NOT NULL;

-- migrate:down
-- Restore NOT NULL constraint (this will fail if there are global scan jobs)
ALTER TABLE scan_jobs ALTER COLUMN bucket_id SET NOT NULL;