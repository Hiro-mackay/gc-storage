-- name: CreateFile :one
INSERT INTO files (
    id, name, folder_id, owner_id, mime_type, size, storage_key, current_version, status, checksum, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING *;

-- name: GetFileByID :one
SELECT * FROM files WHERE id = $1;

-- name: UpdateFile :one
UPDATE files SET
    name = COALESCE(sqlc.narg('name'), name),
    folder_id = COALESCE(sqlc.narg('folder_id'), folder_id),
    size = COALESCE(sqlc.narg('size'), size),
    storage_key = COALESCE(sqlc.narg('storage_key'), storage_key),
    current_version = COALESCE(sqlc.narg('current_version'), current_version),
    status = COALESCE(sqlc.narg('status'), status),
    checksum = COALESCE(sqlc.narg('checksum'), checksum),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateFileStatus :exec
UPDATE files SET status = $2, updated_at = NOW() WHERE id = $1;

-- name: TrashFile :exec
UPDATE files SET status = 'trashed', trashed_at = NOW(), updated_at = NOW() WHERE id = $1;

-- name: RestoreFile :exec
UPDATE files SET status = 'active', trashed_at = NULL, updated_at = NOW() WHERE id = $1;

-- name: DeleteFile :exec
UPDATE files SET status = 'deleted', updated_at = NOW() WHERE id = $1;

-- name: ListFilesByFolderID :many
SELECT * FROM files
WHERE folder_id = $1 AND status = 'active'
ORDER BY name ASC;

-- name: ListFilesByOwnerID :many
SELECT * FROM files
WHERE owner_id = $1 AND status = 'active'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListTrashedFilesByOwnerID :many
SELECT * FROM files
WHERE owner_id = $1 AND status = 'trashed'
ORDER BY trashed_at DESC;

-- name: CountFilesByFolderID :one
SELECT COUNT(*) FROM files
WHERE folder_id = $1 AND status = 'active';

-- name: FileExistsByName :one
SELECT EXISTS(
    SELECT 1 FROM files
    WHERE folder_id = $1 AND owner_id = $2 AND name = $3 AND status != 'deleted'
);

-- name: GetFilesToAutoDelete :many
SELECT * FROM files
WHERE status = 'trashed' AND trashed_at < $1;

-- name: SearchFilesByName :many
SELECT * FROM files
WHERE owner_id = $1 AND name ILIKE '%' || $2 || '%' AND status = 'active'
ORDER BY name ASC
LIMIT $3 OFFSET $4;

-- name: GetFileTotalSizeByOwnerID :one
SELECT COALESCE(SUM(size), 0)::bigint FROM files
WHERE owner_id = $1 AND status = 'active';
