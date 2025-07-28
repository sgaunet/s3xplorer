-- migrate:up
CREATE TABLE buckets (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    region VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE s3_objects (
    id SERIAL PRIMARY KEY,
    bucket_id INTEGER NOT NULL REFERENCES buckets(id) ON DELETE CASCADE,
    key VARCHAR(1024) NOT NULL,
    size BIGINT NOT NULL DEFAULT 0,
    last_modified TIMESTAMP WITH TIME ZONE,
    etag VARCHAR(255),
    storage_class VARCHAR(50),
    is_folder BOOLEAN DEFAULT FALSE,
    prefix VARCHAR(1024),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(bucket_id, key)
);

CREATE TABLE scan_jobs (
    id SERIAL PRIMARY KEY,
    bucket_id INTEGER NOT NULL REFERENCES buckets(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, running, completed, failed
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    objects_scanned INTEGER DEFAULT 0,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_s3_objects_bucket_id ON s3_objects(bucket_id);
CREATE INDEX idx_s3_objects_prefix ON s3_objects(prefix);
CREATE INDEX idx_s3_objects_key ON s3_objects(key);
CREATE INDEX idx_s3_objects_is_folder ON s3_objects(is_folder);
CREATE INDEX idx_s3_objects_last_modified ON s3_objects(last_modified);
CREATE INDEX idx_scan_jobs_bucket_id ON scan_jobs(bucket_id);
CREATE INDEX idx_scan_jobs_status ON scan_jobs(status);

-- migrate:down
DROP TABLE IF EXISTS scan_jobs;
DROP TABLE IF EXISTS s3_objects;
DROP TABLE IF EXISTS buckets;