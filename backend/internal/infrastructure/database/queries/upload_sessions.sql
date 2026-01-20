-- name: CreateUploadSession :one
INSERT INTO upload_sessions (
    id, file_id, owner_id, owner_type, folder_id, file_name, mime_type, total_size,
    storage_key, minio_upload_id, is_multipart, total_parts, uploaded_parts, status,
    created_at, updated_at, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
) RETURNING *;

-- name: GetUploadSessionByID :one
SELECT * FROM upload_sessions WHERE id = $1;

-- name: GetUploadSessionByFileID :one
SELECT * FROM upload_sessions
WHERE file_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: GetUploadSessionByStorageKey :one
SELECT * FROM upload_sessions
WHERE storage_key = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: UpdateUploadSession :one
UPDATE upload_sessions SET
    minio_upload_id = COALESCE(sqlc.narg('minio_upload_id'), minio_upload_id),
    uploaded_parts = COALESCE(sqlc.narg('uploaded_parts'), uploaded_parts),
    status = COALESCE(sqlc.narg('status'), status),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateUploadSessionStatus :exec
UPDATE upload_sessions SET status = $2, updated_at = NOW() WHERE id = $1;

-- name: IncrementUploadedParts :exec
UPDATE upload_sessions
SET uploaded_parts = uploaded_parts + 1, updated_at = NOW()
WHERE id = $1;

-- name: ListExpiredUploadSessions :many
SELECT * FROM upload_sessions
WHERE status IN ('pending', 'in_progress') AND expires_at < NOW();

-- name: DeleteUploadSession :exec
DELETE FROM upload_sessions WHERE id = $1;
