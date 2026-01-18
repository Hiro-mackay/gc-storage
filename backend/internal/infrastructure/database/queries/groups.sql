-- name: CreateGroup :one
INSERT INTO groups (
    id, name, description, owner_id, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetGroupByID :one
SELECT * FROM groups WHERE id = $1;

-- name: UpdateGroup :one
UPDATE groups SET
    name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: TransferGroupOwnership :exec
UPDATE groups SET owner_id = $2, updated_at = NOW() WHERE id = $1;

-- name: DeleteGroup :exec
DELETE FROM groups WHERE id = $1;

-- name: ListGroupsByOwnerID :many
SELECT * FROM groups
WHERE owner_id = $1
ORDER BY created_at DESC;

-- name: ListGroupsByUserID :many
SELECT g.* FROM groups g
INNER JOIN memberships m ON g.id = m.group_id
WHERE m.user_id = $1
ORDER BY g.created_at DESC;
