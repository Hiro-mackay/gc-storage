-- name: CreateUploadPart :one
INSERT INTO upload_parts (
    id, session_id, part_number, size, etag, uploaded_at
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: ListUploadPartsBySessionID :many
SELECT * FROM upload_parts
WHERE session_id = $1
ORDER BY part_number ASC;

-- name: DeleteUploadPartsBySessionID :exec
DELETE FROM upload_parts WHERE session_id = $1;
