-- name: CreateShareLink :one
INSERT INTO share_links (
    id, resource_type, resource_id, token, permission, password_hash, max_downloads, expires_at, is_active, created_by, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING *;

-- name: GetShareLinkByID :one
SELECT * FROM share_links WHERE id = $1;

-- name: GetShareLinkByToken :one
SELECT * FROM share_links WHERE token = $1;

-- name: GetActiveShareLinkByToken :one
SELECT * FROM share_links
WHERE token = $1 AND is_active = TRUE
AND (expires_at IS NULL OR expires_at > NOW())
AND (max_downloads IS NULL OR download_count < max_downloads);

-- name: UpdateShareLink :one
UPDATE share_links SET
    permission = COALESCE(sqlc.narg('permission'), permission),
    password_hash = COALESCE(sqlc.narg('password_hash'), password_hash),
    max_downloads = COALESCE(sqlc.narg('max_downloads'), max_downloads),
    expires_at = COALESCE(sqlc.narg('expires_at'), expires_at),
    is_active = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: IncrementDownloadCount :exec
UPDATE share_links SET download_count = download_count + 1 WHERE id = $1;

-- name: DeactivateShareLink :exec
UPDATE share_links SET is_active = FALSE, updated_at = NOW() WHERE id = $1;

-- name: DeleteShareLink :exec
DELETE FROM share_links WHERE id = $1;

-- name: ListShareLinksByResource :many
SELECT * FROM share_links
WHERE resource_type = $1 AND resource_id = $2
ORDER BY created_at DESC;

-- name: ListShareLinksByCreator :many
SELECT * FROM share_links
WHERE created_by = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListActiveShareLinks :many
SELECT * FROM share_links
WHERE is_active = TRUE
AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY created_at DESC;

-- name: CountShareLinksByResource :one
SELECT COUNT(*) FROM share_links
WHERE resource_type = $1 AND resource_id = $2 AND is_active = TRUE;

-- name: DeactivateExpiredShareLinks :exec
UPDATE share_links SET is_active = FALSE, updated_at = NOW()
WHERE is_active = TRUE AND expires_at IS NOT NULL AND expires_at < NOW();
