-- name: CreateFileVersion :one
INSERT INTO file_versions (
    id, file_id, version_number, minio_version_id, size, checksum, uploaded_by, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetFileVersionByID :one
SELECT * FROM file_versions WHERE id = $1;

-- name: ListFileVersionsByFileID :many
SELECT * FROM file_versions
WHERE file_id = $1
ORDER BY version_number DESC;

-- name: GetFileVersionByFileAndVersion :one
SELECT * FROM file_versions
WHERE file_id = $1 AND version_number = $2;

-- name: GetLatestFileVersion :one
SELECT * FROM file_versions
WHERE file_id = $1
ORDER BY version_number DESC
LIMIT 1;

-- name: DeleteFileVersion :exec
DELETE FROM file_versions WHERE id = $1;

-- name: DeleteFileVersionsByFileID :exec
DELETE FROM file_versions WHERE file_id = $1;

-- name: CountFileVersionsByFileID :one
SELECT COUNT(*) FROM file_versions WHERE file_id = $1;

-- name: ListFileVersionsByFileIDs :many
SELECT * FROM file_versions
WHERE file_id = ANY($1::uuid[])
ORDER BY file_id, version_number DESC;

-- name: CreateFileVersionsBulk :copyfrom
INSERT INTO file_versions (id, file_id, version_number, minio_version_id, size, checksum, uploaded_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetNextVersionNumber :one
SELECT COALESCE(MAX(version_number), 0) + 1 FROM file_versions WHERE file_id = $1;
