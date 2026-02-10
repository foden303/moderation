-- BadImages table for storing pHash of NSFW/bad images
CREATE TABLE IF NOT EXISTS bad_images (
    id BIGSERIAL PRIMARY KEY,
    phash BIGINT NOT NULL,
    category VARCHAR(100) DEFAULT 'nsfw',
    nsfw_score FLOAT DEFAULT 0.0,
    source_url TEXT DEFAULT '',
    added_by VARCHAR(255) DEFAULT 'auto',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for fast pHash lookup
CREATE INDEX idx_bad_images_phash ON bad_images(phash);
CREATE INDEX idx_bad_images_category ON bad_images(category);

-- Unique constraint on phash to prevent duplicates
CREATE UNIQUE INDEX idx_bad_images_phash_unique ON bad_images(phash);
