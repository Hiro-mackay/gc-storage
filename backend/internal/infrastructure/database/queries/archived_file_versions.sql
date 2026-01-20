-- name: CreateArchivedFileVersionsBulk :copyfrom
INSERT INTO archived_file_versions (
    id, archived_file_id, original_version_id, version_number, minio_version_id, size, checksum, uploaded_by, created_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: ListArchivedFileVersionsByArchivedFileID :many
SELECT * FROM archived_file_versions
WHERE archived_file_id = $1
ORDER BY version_number DESC;

-- name: DeleteArchivedFileVersionsByArchivedFileID :exec
DELETE FROM archived_file_versions WHERE archived_file_id = $1;
