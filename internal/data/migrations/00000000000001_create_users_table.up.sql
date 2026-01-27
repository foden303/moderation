CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY,
    nickname TEXT NOT NULL,
    user_identifier TEXT NOT NULL,  -- public key hash or client UUID
    effective_storage_quota BIGINT NOT NULL DEFAULT 2147483648, -- 2 GB default
    storage_photos_used BIGINT NOT NULL DEFAULT 0,
    storage_video_used BIGINT NOT NULL DEFAULT 0,
    storage_document_used BIGINT NOT NULL DEFAULT 0,
    storage_audio_used BIGINT NOT NULL DEFAULT 0,
    storage_compress_used BIGINT NOT NULL DEFAULT 0,
    storage_other_used BIGINT NOT NULL DEFAULT 0,
    storage_total_used BIGINT NOT NULL
    GENERATED ALWAYS AS (storage_photos_used + storage_video_used + storage_document_used + storage_audio_used + storage_compress_used + storage_other_used) STORED,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
