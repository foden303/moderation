CREATE TABLE IF NOT EXISTS file_paths (
    file_id uuid NOT NULL,
    ancestor_id uuid NOT NULL,
    PRIMARY KEY (file_id, ancestor_id)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_file_paths_file_id ON file_paths(file_id, ancestor_id);