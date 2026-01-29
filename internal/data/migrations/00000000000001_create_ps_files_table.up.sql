CREATE TABLE IF NOT EXISTS ps_files (
    id uuid PRIMARY KEY DEFAULT uuidv7(),
    hash TEXT NOT NULL, -- SHA-256 of the file or chunk
    size BIGINT NOT NULL,
    encrypted BOOLEAN NOT NULL DEFAULT TRUE, -- client-side encryption flag
    mime_type TEXT NOT NULL,        -- MIME type, ex: image/png, application/pdf
    ext TEXT NOT NULL, -- file extension, ex: png, pdf
    checksum TEXT,             -- original file checksum (MD5/SHA256)
    total_chunks INT NOT NULL DEFAULT 0,
    reference_count INT NOT NULL DEFAULT 1, -- deduplication reference
    status TEXT NOT NULL DEFAULT 'uploading', -- uploading | completed
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_ps_files_hash ON ps_files(hash);