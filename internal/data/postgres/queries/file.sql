-- name: CreateFile :one
INSERT INTO files (
    folder_id, owner_id, name, type, space, file_hash, file_size, 
    file_type, file_ext, file_mime_type, file_video_resolution,
    platform, shared, favorite, unique_hash, del_signature
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
)
RETURNING *;

-- name: GetFileByID :one
SELECT * FROM files 
WHERE id = $1 AND deleted_at IS NULL 
LIMIT 1;

-- name: GetFileByFileHash :one
SELECT * FROM files
WHERE owner_id = $1 
  AND (folder_id = $2 OR (folder_id IS NULL AND $2 IS NULL))
  AND file_hash = $3
  AND deleted_at IS NULL
LIMIT 1;

-- name: GetFiles :many
SELECT * FROM files
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetFilesByIDs :many
SELECT * FROM files
WHERE owner_id = $1 
  AND id = ANY($2::uuid[])
  AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: GetFoldersByFolderID :many
SELECT * FROM files
WHERE owner_id = $1
  AND (folder_id = $2 OR (folder_id IS NULL AND $2 IS NULL))
  AND deleted_at IS NULL
ORDER BY type DESC, name ASC;

-- ========================================
-- CURSOR PAGINATION
-- ========================================

-- name: GetFilesWithCursor :many
SELECT * FROM files
WHERE owner_id = $1
  AND deleted_at IS NULL
  AND (created_at, id) < ($2, $3)
ORDER BY created_at DESC, id DESC
LIMIT $4;

-- name: GetFilesFirstPage :many
SELECT * FROM files
WHERE owner_id = $1
  AND deleted_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT $2;

-- name: GetFilesByFolderWithCursor :many
SELECT * FROM files
WHERE owner_id = $1
  AND (folder_id = $2 OR (folder_id IS NULL AND $2 IS NULL))
  AND deleted_at IS NULL
  AND (created_at, id) < ($3, $4)
ORDER BY created_at DESC, id DESC
LIMIT $5;

-- name: GetFilesByFolderFirstPage :many
SELECT * FROM files
WHERE owner_id = $1
  AND (folder_id = $2 OR (folder_id IS NULL AND $2 IS NULL))
  AND deleted_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT $3;

-- ========================================
-- TRASH
-- ========================================

-- name: GetAllFilesInTrash :many
SELECT * FROM files
WHERE owner_id = $1 
  AND deleted_at IS NOT NULL
ORDER BY deleted_at DESC;

-- name: GetFilesExpiredInTrash :many
SELECT * FROM files
WHERE deleted_at IS NOT NULL
  AND deleted_at < $1
ORDER BY deleted_at ASC
LIMIT $2;

-- name: GetFilesByIDsUnscoped :many
SELECT * FROM files
WHERE owner_id = $1 
  AND id = ANY($2::uuid[])
ORDER BY created_at DESC;

-- name: GetFilesByIDsInTrash :many
SELECT * FROM files
WHERE owner_id = $1 
  AND id = ANY($2::uuid[])
  AND deleted_at IS NOT NULL
ORDER BY deleted_at DESC;

-- ========================================
-- DELETE
-- ========================================

-- name: SoftDeleteFiles :exec
UPDATE files
SET deleted_at = now(), deleted_by = $1, updated_at = now()
WHERE id = ANY($2::uuid[])
  AND deleted_at IS NULL;

-- name: DeleteFilesPermanently :exec
DELETE FROM files
WHERE id = ANY($1::uuid[]);

-- ========================================
-- RESTORE
-- ========================================

-- name: RestoreFiles :exec
UPDATE files
SET deleted_at = NULL, deleted_by = NULL, updated_at = now()
WHERE owner_id = $1 
  AND id = ANY($2::uuid[])
  AND deleted_at IS NOT NULL;

-- name: RestoreFilesToRoot :exec
UPDATE files
SET deleted_at = NULL, deleted_by = NULL, folder_id = NULL, updated_at = now()
WHERE owner_id = $1 
  AND id = ANY($2::uuid[])
  AND deleted_at IS NOT NULL;

-- ========================================
-- MOVE
-- ========================================

-- name: MoveFilesToFolder :exec
UPDATE files
SET folder_id = $1, updated_at = now()
WHERE owner_id = $2 
  AND id = ANY($3::uuid[])
  AND deleted_at IS NULL;

-- ========================================
-- UPDATE
-- ========================================

-- name: UpdateFileNameByID :exec
UPDATE files
SET name = $1, updated_at = now()
WHERE owner_id = $2 
  AND id = $3
  AND deleted_at IS NULL;

-- name: UpdateFavoriteFilesByIDs :exec
UPDATE files
SET favorite = $1, updated_at = now()
WHERE owner_id = $2 
  AND id = ANY($3::uuid[])
  AND deleted_at IS NULL;

-- name: UpdateRecentAccessedAtByIDs :exec
UPDATE files
SET recent_accessed_at = $1, updated_at = now()
WHERE owner_id = $2 
  AND id = ANY($3::uuid[])
  AND deleted_at IS NULL;

-- ========================================
-- COUNT
-- ========================================

-- name: CountFilesByOwner :one
SELECT COUNT(*) FROM files
WHERE owner_id = $1 AND deleted_at IS NULL;

-- name: CountFilesInTrash :one
SELECT COUNT(*) FROM files
WHERE owner_id = $1 AND deleted_at IS NOT NULL;
