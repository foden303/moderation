-- name: UpsertImageCache :one
INSERT INTO image_caches (
    file_hash, phash, detect_result,
    category, nsfw_score, model_version, source_url, added_by, expires_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (file_hash) DO UPDATE SET
    detect_result = EXCLUDED.detect_result,
    category = EXCLUDED.category,
    nsfw_score = EXCLUDED.nsfw_score,
    model_version = EXCLUDED.model_version,
    updated_at = NOW()
RETURNING *;

-- name: GetImageCache :one
SELECT * FROM image_caches WHERE file_hash = $1;

-- name: FindSimilarByPHash :many
SELECT *,
       bit_count((phash # @target_phash)::bit(64))::integer AS distance
FROM image_caches
WHERE category <> 'safe'
  AND bit_count((phash # @target_phash)::bit(64))::integer <= @max_distance
ORDER BY distance ASC
LIMIT 5;

-- name: DeleteImageCache :exec
DELETE FROM image_caches WHERE file_hash = $1;

-- name: DeleteExpiredImageCaches :execrows
DELETE FROM image_caches WHERE expires_at IS NOT NULL AND expires_at < NOW();

-- name: CountImageCaches :one
SELECT COUNT(*) FROM image_caches;

-- name: ListImageCachesByCategory :many
SELECT * FROM image_caches
WHERE category = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListImageCaches :many
SELECT * FROM image_caches
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountImageCachesByCategory :one
SELECT COUNT(*) FROM image_caches WHERE category = $1;

-- name: GetAllImageCaches :many
SELECT * FROM image_caches;
