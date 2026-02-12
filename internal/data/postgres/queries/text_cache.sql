-- name: UpsertTextCache :one
INSERT INTO text_caches (
    content_hash, normalized_content, detect_result,
    category, nsfw_score, model_version, added_by, expires_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (content_hash) DO UPDATE SET
    detect_result = EXCLUDED.detect_result,
    category = EXCLUDED.category,
    nsfw_score = EXCLUDED.nsfw_score,
    model_version = EXCLUDED.model_version,
    updated_at = NOW()
RETURNING *;

-- name: GetTextCache :one
SELECT * FROM text_caches WHERE content_hash = $1;

-- name: DeleteTextCache :exec
DELETE FROM text_caches WHERE content_hash = $1;

-- name: DeleteExpiredTextCaches :execrows
DELETE FROM text_caches WHERE expires_at IS NOT NULL AND expires_at < NOW();

-- name: CountTextCaches :one
SELECT COUNT(*) FROM text_caches;

-- name: ListTextCachesByCategory :many
SELECT * FROM text_caches
WHERE category = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListTextCaches :many
SELECT * FROM text_caches
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountTextCachesByCategory :one
SELECT COUNT(*) FROM text_caches WHERE category = $1;

-- name: GetAllTextCaches :many
SELECT * FROM text_caches;
