-- BadWords table for content moderation
CREATE TABLE IF NOT EXISTS bad_words (
    id BIGSERIAL PRIMARY KEY,
    word VARCHAR(255) NOT NULL UNIQUE,
    category VARCHAR(100) DEFAULT '',
    severity INT DEFAULT 1,
    added_by VARCHAR(255) DEFAULT '',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for fast word lookup
CREATE INDEX idx_bad_words_word ON bad_words(word);
CREATE INDEX idx_bad_words_category ON bad_words(category);
