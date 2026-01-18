-- name: CreateShareLinkAccess :one
INSERT INTO share_link_accesses (
    id, share_link_id, accessed_by, ip_address, user_agent, action, accessed_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: ListShareLinkAccessesByLinkID :many
SELECT * FROM share_link_accesses
WHERE share_link_id = $1
ORDER BY accessed_at DESC
LIMIT $2 OFFSET $3;

-- name: CountShareLinkAccesses :one
SELECT COUNT(*) FROM share_link_accesses WHERE share_link_id = $1;

-- name: CountShareLinkDownloads :one
SELECT COUNT(*) FROM share_link_accesses
WHERE share_link_id = $1 AND action = 'download';

-- name: GetRecentShareLinkAccesses :many
SELECT * FROM share_link_accesses
WHERE share_link_id = $1 AND accessed_at > $2
ORDER BY accessed_at DESC;

-- name: DeleteOldShareLinkAccesses :exec
DELETE FROM share_link_accesses WHERE accessed_at < $1;

-- name: AnonymizeOldShareLinkAccesses :exec
UPDATE share_link_accesses SET
    ip_address = NULL,
    user_agent = NULL
WHERE accessed_at < $1 AND ip_address IS NOT NULL;
