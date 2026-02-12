-- name: CreateTextDataset :one
INSERT INTO text_datasets (
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

-- name: GetTextDataset :one
SELECT * FROM text_datasets WHERE content_hash = $1;

-- name: ListTextDatasets :many
SELECT * FROM text_datasets
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListTextDatasetsByCategory :many
SELECT * FROM text_datasets
WHERE category = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountTextDatasets :one
SELECT COUNT(*) FROM text_datasets;

-- name: CountTextDatasetsByCategory :one
SELECT COUNT(*) FROM text_datasets WHERE category = $1;

-- name: DeleteTextDataset :exec
DELETE FROM text_datasets WHERE content_hash = $1;

-- name: GetAllTextDatasets :many
SELECT * FROM text_datasets;

-- name: GetTextDatasetByID :one
SELECT * FROM text_datasets WHERE id = $1;

-- name: UpdateTextDataset :one
UPDATE text_datasets
SET category = $2, nsfw_score = $3, added_by = $4, updated_at = NOW()
WHERE id = $1
RETURNING *;
