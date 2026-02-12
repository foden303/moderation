-- create text datasets table
CREATE TABLE text_datasets (
    id BIGSERIAL PRIMARY KEY,
    content_hash TEXT NOT NULL UNIQUE,
    normalized_content TEXT NOT NULL,
    detect_result JSONB NOT NULL,
    category TEXT NOT NULL DEFAULT 'safe',
    nsfw_score DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    model_version TEXT NOT NULL,
    added_by TEXT NOT NULL DEFAULT 'manual',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- unique index for normalized content
CREATE UNIQUE INDEX uq_text_datasets_normalized
ON text_datasets(normalized_content);
-- index for category
CREATE INDEX idx_text_datasets_category
ON text_datasets(category);

-- create text caches table
CREATE TABLE text_caches (
    content_hash TEXT PRIMARY KEY,
    normalized_content TEXT NOT NULL,
    detect_result JSONB NOT NULL,
    category TEXT NOT NULL DEFAULT 'safe',
    nsfw_score DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    model_version TEXT NOT NULL,
    added_by TEXT NOT NULL DEFAULT 'manual',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- index for expiry
CREATE INDEX idx_text_caches_expiry
ON text_caches(expires_at)
WHERE expires_at IS NOT NULL;
-- index for detect result
CREATE INDEX idx_text_caches_detect_result
ON text_caches USING GIN (detect_result);

-- create image datasets table
CREATE TABLE image_datasets (
    id BIGSERIAL PRIMARY KEY,
    file_hash TEXT NOT NULL UNIQUE,
    phash BIGINT NOT NULL,
    detect_result JSONB NOT NULL,
    category TEXT NOT NULL DEFAULT 'safe',
    nsfw_score DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    model_version TEXT NOT NULL,
    source_url TEXT NOT NULL DEFAULT '',
    added_by TEXT NOT NULL DEFAULT 'manual',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- index for phash
CREATE INDEX idx_image_datasets_phash
ON image_datasets(phash);
-- index for category
CREATE INDEX idx_image_datasets_category
ON image_datasets(category);

-- create image caches table
CREATE TABLE image_caches (
    file_hash TEXT PRIMARY KEY,
    phash BIGINT NOT NULL,
    detect_result JSONB NOT NULL,
    category TEXT NOT NULL DEFAULT 'safe',
    nsfw_score DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    model_version TEXT NOT NULL,
    source_url TEXT NOT NULL DEFAULT '',
    added_by TEXT NOT NULL DEFAULT 'manual',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- index for phash
CREATE INDEX idx_image_caches_phash
ON image_caches(phash);

-- index for expiry
CREATE INDEX idx_image_caches_expiry
ON image_caches(expires_at)
WHERE expires_at IS NOT NULL;

-- index for detect result
CREATE INDEX idx_image_caches_detect_result
ON image_caches USING GIN (detect_result);


CREATE OR REPLACE FUNCTION sync_text_cache_to_dataset()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.category <> 'safe' THEN

        INSERT INTO text_datasets (
            content_hash,
            normalized_content,
            detect_result,
            category,
            nsfw_score,
            model_version,
            added_by
        )
        VALUES (
            NEW.content_hash,
            NEW.normalized_content,
            NEW.detect_result,
            NEW.category,
            NEW.nsfw_score,
            NEW.model_version,
            NEW.added_by
        )
        ON CONFLICT (content_hash) DO NOTHING;

    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER trg_text_cache_sync
AFTER INSERT OR UPDATE ON text_caches
FOR EACH ROW
EXECUTE FUNCTION sync_text_cache_to_dataset();



CREATE OR REPLACE FUNCTION sync_image_cache_to_dataset()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.category <> 'safe' THEN

        INSERT INTO image_datasets (
            file_hash,
            phash,
            detect_result,
            category,
            nsfw_score,
            model_version,
            source_url,
            added_by
        )
        VALUES (
            NEW.file_hash,
            NEW.phash,
            NEW.detect_result,
            NEW.category,
            NEW.nsfw_score,
            NEW.model_version,
            NEW.source_url,
            NEW.added_by
        )
        ON CONFLICT (file_hash) DO NOTHING;

    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;


CREATE TRIGGER trg_image_cache_sync
AFTER INSERT OR UPDATE ON image_caches
FOR EACH ROW
EXECUTE FUNCTION sync_image_cache_to_dataset();





