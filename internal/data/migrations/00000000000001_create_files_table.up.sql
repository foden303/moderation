CREATE TABLE IF NOT EXISTS files (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    folder_id uuid,
    owner_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'file',
    space TEXT NOT NULL DEFAULT 'normal', -- normal | privacy
    file_hash TEXT,
    file_size BIGINT,
    file_type TEXT, -- image | video | document | audio | compressed | other
    file_ext TEXT,
    file_mime_type TEXT,
    file_video_resolution TEXT,
    recent_accessed_at TIMESTAMPTZ NULL,
    shared BOOLEAN NOT NULL DEFAULT FALSE,  -- flag whether shared or not
    favorite BOOLEAN NOT NULL DEFAULT FALSE, -- flag whether favorite or not
    platform BIGINT NOT NULL DEFAULT 1, -- platform: 1-IM, 2-Web, 3-Desktop, 4-Mobile
    unique_hash TEXT, -- unique hash for deduplication
    del_signature TEXT, -- digital signature for file integrity
    deleted_at TIMESTAMPTZ NULL,     -- Trash
    deleted_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_files_owner_deleted_at ON files (owner_id, deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_uniq_files_alive_hash_per_folder ON files (owner_id, folder_id, file_hash,name)
WHERE deleted_at IS NULL AND file_hash IS NOT NULL;