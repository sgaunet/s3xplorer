-- migrate:up
ALTER TABLE s3_objects ADD COLUMN marked_for_deletion BOOLEAN DEFAULT FALSE;

-- Add index for efficient deletion queries
CREATE INDEX idx_s3_objects_marked_for_deletion ON s3_objects(marked_for_deletion);

-- migrate:down
DROP INDEX IF EXISTS idx_s3_objects_marked_for_deletion;
ALTER TABLE s3_objects DROP COLUMN IF EXISTS marked_for_deletion;