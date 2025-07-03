-- migrate:up
-- Add bucket lifecycle management columns for bucket-level synchronization
ALTER TABLE buckets ADD COLUMN marked_for_deletion BOOLEAN DEFAULT FALSE;
ALTER TABLE buckets ADD COLUMN last_accessible_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE buckets ADD COLUMN access_error TEXT;

-- Add indexes for efficient bucket lifecycle queries
CREATE INDEX idx_buckets_marked_for_deletion ON buckets(marked_for_deletion);
CREATE INDEX idx_buckets_last_accessible_at ON buckets(last_accessible_at);

-- migrate:down
-- Remove bucket lifecycle management columns
DROP INDEX IF EXISTS idx_buckets_last_accessible_at;
DROP INDEX IF EXISTS idx_buckets_marked_for_deletion;
ALTER TABLE buckets DROP COLUMN IF EXISTS access_error;
ALTER TABLE buckets DROP COLUMN IF EXISTS last_accessible_at;
ALTER TABLE buckets DROP COLUMN IF EXISTS marked_for_deletion;