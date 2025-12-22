-- migrate:up

-- Primary composite index for folder navigation queries
-- Benefits: ListS3Objects, ListS3Folders, ListS3Files, CountS3Objects,
--           CountDirectChildrenFolders, CountDirectChildrenFiles, GetDirectChildren
-- This index uses PostgreSQL's "leftmost prefix" rule, so it can be used for:
--   - bucket_id only
--   - bucket_id + prefix
--   - bucket_id + prefix + is_folder (optimal)
CREATE INDEX idx_s3_objects_bucket_prefix_folder
  ON s3_objects(bucket_id, prefix, is_folder);

-- Secondary composite index for breadcrumb navigation
-- Benefits: GetBreadcrumbPath query (finds all parent folders in a path)
CREATE INDEX idx_s3_objects_bucket_folder
  ON s3_objects(bucket_id, is_folder);

-- Partial index for deletion sync queries
-- Benefits: DeleteMarkedObjects (only indexes rows marked for deletion)
-- This partial index is much smaller and faster than a full index
CREATE INDEX idx_s3_objects_bucket_deletion
  ON s3_objects(bucket_id, marked_for_deletion)
  WHERE marked_for_deletion = TRUE;

-- migrate:down

-- Drop indexes in reverse order of creation
DROP INDEX IF EXISTS idx_s3_objects_bucket_deletion;
DROP INDEX IF EXISTS idx_s3_objects_bucket_folder;
DROP INDEX IF EXISTS idx_s3_objects_bucket_prefix_folder;
