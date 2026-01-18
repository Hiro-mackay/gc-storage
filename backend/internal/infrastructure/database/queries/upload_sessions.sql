-- name: CreateUploadSession :one
INSERT INTO upload_sessions (
    id, file_id, upload_id, status, total_parts, completed_parts, expires_at, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetUploadSessionByID :one
SELECT * FROM upload_sessions WHERE id = $1;

-- name: GetActiveUploadSessionByFileID :one
SELECT * FROM upload_sessions
WHERE file_id = $1 AND status IN ('initiated', 'uploading', 'completing')
ORDER BY created_at DESC
LIMIT 1;

-- name: UpdateUploadSession :one
UPDATE upload_sessions SET
    upload_id = COALESCE(sqlc.narg('upload_id'), upload_id),
    status = COALESCE(sqlc.narg('status'), status),
    total_parts = COALESCE(sqlc.narg('total_parts'), total_parts),
    completed_parts = COALESCE(sqlc.narg('completed_parts'), completed_parts),
    completed_at = COALESCE(sqlc.narg('completed_at'), completed_at)
WHERE id = $1
RETURNING *;

-- name: IncrementCompletedParts :exec
UPDATE upload_sessions SET completed_parts = completed_parts + 1 WHERE id = $1;

-- name: CompleteUploadSession :exec
UPDATE upload_sessions SET status = 'completed', completed_at = NOW() WHERE id = $1;

-- name: FailUploadSession :exec
UPDATE upload_sessions SET status = 'failed' WHERE id = $1;

-- name: AbortUploadSession :exec
UPDATE upload_sessions SET status = 'aborted' WHERE id = $1;

-- name: ListExpiredUploadSessions :many
SELECT * FROM upload_sessions
WHERE status IN ('initiated', 'uploading') AND expires_at < NOW();

-- name: DeleteExpiredUploadSessions :exec
DELETE FROM upload_sessions
WHERE status IN ('completed', 'failed', 'aborted') AND created_at < $1;
