-- name: CreateFileVersion :one
INSERT INTO file_versions (
    id, file_id, version_number, size, storage_key, checksum, created_by, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetFileVersionByID :one
SELECT * FROM file_versions WHERE id = $1;

-- name: GetFileVersion :one
SELECT * FROM file_versions
WHERE file_id = $1 AND version_number = $2;

-- name: GetLatestFileVersion :one
SELECT * FROM file_versions
WHERE file_id = $1
ORDER BY version_number DESC
LIMIT 1;

-- name: ListFileVersions :many
SELECT * FROM file_versions
WHERE file_id = $1
ORDER BY version_number DESC;

-- name: DeleteFileVersion :exec
DELETE FROM file_versions WHERE id = $1;

-- name: DeleteFileVersionsByFileID :exec
DELETE FROM file_versions WHERE file_id = $1;

-- name: CountFileVersions :one
SELECT COUNT(*) FROM file_versions WHERE file_id = $1;

-- name: GetNextVersionNumber :one
SELECT COALESCE(MAX(version_number), 0) + 1 FROM file_versions WHERE file_id = $1;
