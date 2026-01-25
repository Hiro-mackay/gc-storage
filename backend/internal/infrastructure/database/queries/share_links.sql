-- name: CreateShareLink :one
INSERT INTO share_links (
    id, token, resource_type, resource_id, created_by, permission,
    password_hash, expires_at, max_access_count, access_count, status, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
) RETURNING *;

-- name: GetShareLinkByID :one
SELECT * FROM share_links WHERE id = $1;

-- name: GetShareLinkByToken :one
SELECT * FROM share_links WHERE token = $1;

-- name: UpdateShareLink :one
UPDATE share_links SET
    permission = COALESCE(sqlc.narg('permission'), permission),
    password_hash = COALESCE(sqlc.narg('password_hash'), password_hash),
    expires_at = COALESCE(sqlc.narg('expires_at'), expires_at),
    max_access_count = COALESCE(sqlc.narg('max_access_count'), max_access_count),
    access_count = COALESCE(sqlc.narg('access_count'), access_count),
    status = COALESCE(sqlc.narg('status'), status),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteShareLink :exec
DELETE FROM share_links WHERE id = $1;

-- name: ListShareLinksByResource :many
SELECT * FROM share_links
WHERE resource_type = $1 AND resource_id = $2
ORDER BY created_at DESC;

-- name: ListActiveShareLinksByResource :many
SELECT * FROM share_links
WHERE resource_type = $1 AND resource_id = $2 AND status = 'active'
ORDER BY created_at DESC;

-- name: ListShareLinksByCreator :many
SELECT * FROM share_links
WHERE created_by = $1
ORDER BY created_at DESC;

-- name: ListExpiredShareLinks :many
SELECT * FROM share_links
WHERE status = 'active' AND expires_at IS NOT NULL AND expires_at < NOW();

-- name: UpdateShareLinksStatusBatch :execrows
UPDATE share_links
SET status = $2, updated_at = NOW()
WHERE id = ANY($1::uuid[]);

-- name: IncrementShareLinkAccessCount :exec
UPDATE share_links SET access_count = access_count + 1, updated_at = NOW() WHERE id = $1;
