-- migrate:up
ALTER TABLE scan_jobs ADD COLUMN objects_deleted INTEGER DEFAULT 0;
ALTER TABLE scan_jobs ADD COLUMN objects_updated INTEGER DEFAULT 0;
ALTER TABLE scan_jobs ADD COLUMN objects_created INTEGER DEFAULT 0;

-- migrate:down
ALTER TABLE scan_jobs DROP COLUMN IF EXISTS objects_deleted;
ALTER TABLE scan_jobs DROP COLUMN IF EXISTS objects_updated;
ALTER TABLE scan_jobs DROP COLUMN IF EXISTS objects_created;