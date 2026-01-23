-- name: CreateFile :one
INSERT INTO files (
    id, folder_id, owner_id, created_by, name, mime_type, size, storage_key, current_version, status, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING *;

-- name: GetFileByID :one
SELECT * FROM files WHERE id = $1;

-- name: UpdateFile :one
UPDATE files SET
    folder_id = COALESCE(sqlc.narg('folder_id'), folder_id),
    owner_id = COALESCE(sqlc.narg('owner_id'), owner_id),
    name = COALESCE(sqlc.narg('name'), name),
    size = COALESCE(sqlc.narg('size'), size),
    current_version = COALESCE(sqlc.narg('current_version'), current_version),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateFileStatus :exec
UPDATE files SET status = $2, updated_at = NOW() WHERE id = $1;

-- name: DeleteFile :exec
DELETE FROM files WHERE id = $1;

-- name: DeleteFilesBulk :exec
DELETE FROM files WHERE id = ANY($1::uuid[]);

-- name: ListFilesByFolderID :many
SELECT * FROM files
WHERE folder_id = $1 AND status = 'active'
ORDER BY name ASC;

-- name: ListFilesByOwner :many
SELECT * FROM files
WHERE owner_id = $1 AND status = 'active'
ORDER BY created_at DESC;

-- name: ListFilesByCreatedBy :many
SELECT * FROM files
WHERE created_by = $1 AND status = 'active'
ORDER BY created_at DESC;

-- name: GetFileByNameAndFolder :one
SELECT * FROM files
WHERE folder_id = $1 AND name = $2 AND status = 'active'
LIMIT 1;

-- name: GetFileByStorageKey :one
SELECT * FROM files WHERE storage_key = $1;

-- name: FileExistsByNameAndFolder :one
-- Check for both 'uploading' and 'active' status (allow overwriting 'upload_failed' files)
SELECT EXISTS(
    SELECT 1 FROM files
    WHERE folder_id = $1 AND name = $2 AND status IN ('uploading', 'active')
);

-- name: ListFilesByFolderIDs :many
SELECT * FROM files
WHERE folder_id = ANY($1::uuid[]) AND status = 'active';

-- name: ListUploadFailedFiles :many
SELECT * FROM files
WHERE status = 'upload_failed';

-- name: CountFilesByFolderID :one
SELECT COUNT(*) FROM files
WHERE folder_id = $1 AND status = 'active';

-- name: GetFileTotalSizeByOwner :one
SELECT COALESCE(SUM(size), 0)::bigint FROM files
WHERE owner_id = $1 AND status = 'active';

-- name: TransferFileOwnership :exec
UPDATE files SET owner_id = $2, updated_at = NOW() WHERE id = $1;
