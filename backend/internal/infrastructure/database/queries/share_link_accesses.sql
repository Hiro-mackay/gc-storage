-- name: CreateShareLinkAccess :one
INSERT INTO share_link_accesses (
    id, share_link_id, accessed_at, ip_address, user_agent, user_id, action
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetShareLinkAccessByID :one
SELECT * FROM share_link_accesses WHERE id = $1;

-- name: ListShareLinkAccessesByLinkID :many
SELECT * FROM share_link_accesses
WHERE share_link_id = $1
ORDER BY accessed_at DESC;

-- name: ListShareLinkAccessesByLinkIDWithPagination :many
SELECT * FROM share_link_accesses
WHERE share_link_id = $1
ORDER BY accessed_at DESC
LIMIT $2 OFFSET $3;

-- name: CountShareLinkAccessesByLinkID :one
SELECT COUNT(*) FROM share_link_accesses WHERE share_link_id = $1;

-- name: DeleteShareLinkAccessesByLinkID :exec
DELETE FROM share_link_accesses WHERE share_link_id = $1;
