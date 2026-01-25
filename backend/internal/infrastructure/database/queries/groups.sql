-- name: CreateGroup :one
INSERT INTO groups (
    id, name, description, owner_id, status, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetGroupByID :one
SELECT * FROM groups WHERE id = $1;

-- name: UpdateGroup :one
UPDATE groups SET
    name = COALESCE(sqlc.narg('name'), name),
    description = COALESCE(sqlc.narg('description'), description),
    owner_id = COALESCE(sqlc.narg('owner_id'), owner_id),
    status = COALESCE(sqlc.narg('status'), status),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteGroup :exec
DELETE FROM groups WHERE id = $1;

-- name: SoftDeleteGroup :exec
UPDATE groups SET status = 'deleted', updated_at = NOW() WHERE id = $1;

-- name: ListGroupsByOwnerID :many
SELECT * FROM groups WHERE owner_id = $1 ORDER BY created_at DESC;

-- name: ListActiveGroupsByOwnerID :many
SELECT * FROM groups WHERE owner_id = $1 AND status = 'active' ORDER BY created_at DESC;

-- name: ListGroupsByMemberID :many
SELECT g.* FROM groups g
INNER JOIN memberships m ON g.id = m.group_id
WHERE m.user_id = $1 AND g.status = 'active'
ORDER BY g.created_at DESC;

-- name: GroupExistsByID :one
SELECT EXISTS(SELECT 1 FROM groups WHERE id = $1);
