-- name: CreateBadImage :one
INSERT INTO bad_images (phash, category, nsfw_score, source_url, added_by)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (phash) DO UPDATE SET
    nsfw_score = EXCLUDED.nsfw_score,
    updated_at = NOW()
RETURNING *;

-- name: GetBadImageByPHash :one
SELECT * FROM bad_images WHERE phash = $1;

-- name: GetBadImagesByPHashes :many
SELECT * FROM bad_images WHERE phash = ANY($1::bigint[]);

-- name: ListAllBadImages :many
SELECT * FROM bad_images;

-- name: ListBadImagesByCategory :many
SELECT * FROM bad_images
WHERE category = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountBadImages :one
SELECT COUNT(*) FROM bad_images;

-- name: CountBadImagesByCategory :one
SELECT COUNT(*) FROM bad_images WHERE category = $1;

-- name: DeleteBadImage :exec
DELETE FROM bad_images WHERE phash = $1;
