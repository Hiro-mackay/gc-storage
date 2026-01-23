-- name: CreateArchivedFile :one
INSERT INTO archived_files (
    id, original_file_id, original_folder_id, original_path, name, mime_type, size,
    owner_id, created_by, storage_key, archived_at, archived_by, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
) RETURNING *;

-- name: GetArchivedFileByID :one
SELECT * FROM archived_files WHERE id = $1;

-- name: GetArchivedFileByOriginalFileID :one
SELECT * FROM archived_files WHERE original_file_id = $1;

-- name: ListArchivedFilesByOwner :many
SELECT * FROM archived_files
WHERE owner_id = $1
ORDER BY archived_at DESC;

-- name: ListArchivedFilesByOwnerWithPagination :many
SELECT * FROM archived_files
WHERE owner_id = $1
  AND (
    sqlc.narg('cursor_id')::uuid IS NULL
    OR id < sqlc.narg('cursor_id')::uuid
  )
ORDER BY archived_at DESC, id DESC
LIMIT $2;

-- name: ListExpiredArchivedFiles :many
SELECT * FROM archived_files
WHERE expires_at < NOW();

-- name: DeleteArchivedFile :exec
DELETE FROM archived_files WHERE id = $1;
