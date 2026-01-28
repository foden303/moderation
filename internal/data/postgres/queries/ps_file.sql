-- name: CreatePsFile :one
INSERT INTO ps_files (
  hash,
  size,
  encrypted,
  mime_type,
  ext,
  checksum,
  total_chunks,
  reference_count,
  status
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: GetPsFileByHash :one
SELECT * FROM ps_files
WHERE hash = $1 AND status = $2 LIMIT 1;

-- name: CheckPsFileExistsByHash :one
SELECT EXISTS (
  SELECT 1 FROM ps_files
  WHERE hash = $1
) AS exists;