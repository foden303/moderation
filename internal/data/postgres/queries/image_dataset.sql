-- name: CreateImageDataset :one
INSERT INTO image_datasets (
    file_hash, phash, detect_result,
    category, nsfw_score, model_version, source_url, added_by, expires_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (file_hash) DO UPDATE SET
    detect_result = EXCLUDED.detect_result,
    category = EXCLUDED.category,
    nsfw_score = EXCLUDED.nsfw_score,
    model_version = EXCLUDED.model_version,
    source_url = EXCLUDED.source_url,
    updated_at = NOW()
RETURNING *;

-- name: GetImageDataset :one
SELECT * FROM image_datasets WHERE file_hash = $1;

-- name: GetImageDatasetByPHash :many
SELECT * FROM image_datasets WHERE phash = $1;

-- name: ListImageDatasets :many
SELECT * FROM image_datasets
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListImageDatasetsByCategory :many
SELECT * FROM image_datasets
WHERE category = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountImageDatasets :one
SELECT COUNT(*) FROM image_datasets;

-- name: CountImageDatasetsByCategory :one
SELECT COUNT(*) FROM image_datasets WHERE category = $1;

-- name: DeleteImageDataset :exec
DELETE FROM image_datasets WHERE file_hash = $1;

-- name: GetAllImageDatasets :many
SELECT * FROM image_datasets;

-- name: GetImageDatasetByID :one
SELECT * FROM image_datasets WHERE id = $1;

-- name: UpdateImageDataset :one
UPDATE image_datasets
SET category = $2, nsfw_score = $3, source_url = $4, added_by = $5, updated_at = NOW()
WHERE id = $1
RETURNING *;
