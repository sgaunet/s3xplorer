-- migrate:up
-- Add bucket sync statistics to scan jobs for tracking bucket validation and cleanup
ALTER TABLE scan_jobs ADD COLUMN buckets_validated INTEGER DEFAULT 0;
ALTER TABLE scan_jobs ADD COLUMN buckets_marked_inaccessible INTEGER DEFAULT 0;
ALTER TABLE scan_jobs ADD COLUMN buckets_cleaned_up INTEGER DEFAULT 0;
ALTER TABLE scan_jobs ADD COLUMN bucket_validation_errors INTEGER DEFAULT 0;

-- migrate:down
-- Remove bucket sync statistics from scan jobs
ALTER TABLE scan_jobs DROP COLUMN IF EXISTS bucket_validation_errors;
ALTER TABLE scan_jobs DROP COLUMN IF EXISTS buckets_cleaned_up;
ALTER TABLE scan_jobs DROP COLUMN IF EXISTS buckets_marked_inaccessible;
ALTER TABLE scan_jobs DROP COLUMN IF EXISTS buckets_validated;