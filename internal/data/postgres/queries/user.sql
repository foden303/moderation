-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (
  id, nickname, user_identifier, storage_photos_used, storage_video_used, storage_document_used, storage_audio_used, storage_compress_used, storage_other_used
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: UpsertUser :one
INSERT INTO users (
  id, nickname, user_identifier, storage_photos_used, storage_video_used, storage_document_used, storage_audio_used, storage_compress_used, storage_other_used
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9
)
ON CONFLICT (id) DO UPDATE SET
    nickname = EXCLUDED.nickname,
    user_identifier = EXCLUDED.user_identifier,
    updated_at = now()
RETURNING *;

-- name: UpdateUserStorage :exec
UPDATE users
SET
    storage_photos_used = $2,
    storage_video_used = $3,
    storage_document_used = $4,
    storage_audio_used = $5,
    storage_compress_used = $6,
    storage_other_used = $7,
    updated_at = now()
WHERE id = $1;

-- name: UpdateStoragePhotosUsed :exec
UPDATE users SET storage_photos_used = storage_photos_used + $2, updated_at = now() WHERE id = $1;

-- name: UpdateStorageVideoUsed :exec
UPDATE users SET storage_video_used = storage_video_used + $2, updated_at = now() WHERE id = $1;

-- name: UpdateStorageDocumentUsed :exec
UPDATE users SET storage_document_used = storage_document_used + $2, updated_at = now() WHERE id = $1;

-- name: UpdateStorageAudioUsed :exec
UPDATE users SET storage_audio_used = storage_audio_used + $2, updated_at = now() WHERE id = $1;

-- name: UpdateStorageCompressUsed :exec
UPDATE users SET storage_compress_used = storage_compress_used + $2, updated_at = now() WHERE id = $1;

-- name: UpdateStorageOtherUsed :exec
UPDATE users SET storage_other_used = storage_other_used + $2, updated_at = now() WHERE id = $1;
