-- name: CreateBadWord :one
INSERT INTO bad_words (word, category, severity, added_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetBadWordByID :one
SELECT * FROM bad_words WHERE id = $1;

-- name: GetBadWordByWord :one
SELECT * FROM bad_words WHERE word = $1;

-- name: ListBadWords :many
SELECT * FROM bad_words
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListBadWordsByCategory :many
SELECT * FROM bad_words
WHERE category = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListAllBadWords :many
SELECT * FROM bad_words;

-- name: UpdateBadWord :one
UPDATE bad_words
SET word = $2, category = $3, severity = $4, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteBadWord :exec
DELETE FROM bad_words WHERE word = $1;

-- name: CountBadWords :one
SELECT COUNT(*) FROM bad_words;

-- name: CountBadWordsByCategory :one
SELECT COUNT(*) FROM bad_words WHERE category = $1;
