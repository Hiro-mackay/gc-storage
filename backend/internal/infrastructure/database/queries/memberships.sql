-- name: CreateMembership :one
INSERT INTO memberships (
    id, group_id, user_id, role, joined_at
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetMembershipByID :one
SELECT * FROM memberships WHERE id = $1;

-- name: UpdateMembership :one
UPDATE memberships SET
    role = COALESCE(sqlc.narg('role'), role)
WHERE id = $1
RETURNING *;

-- name: DeleteMembership :exec
DELETE FROM memberships WHERE id = $1;

-- name: ListMembershipsByGroupID :many
SELECT * FROM memberships WHERE group_id = $1 ORDER BY joined_at ASC;

-- name: ListMembershipsByGroupIDWithUsers :many
SELECT
    m.id, m.group_id, m.user_id, m.role, m.joined_at,
    u.email, u.display_name, u.status as user_status
FROM memberships m
INNER JOIN users u ON m.user_id = u.id
WHERE m.group_id = $1
ORDER BY m.joined_at ASC;

-- name: ListMembershipsByUserID :many
SELECT * FROM memberships WHERE user_id = $1;

-- name: GetMembershipByGroupAndUser :one
SELECT * FROM memberships WHERE group_id = $1 AND user_id = $2;

-- name: MembershipExists :one
SELECT EXISTS(SELECT 1 FROM memberships WHERE group_id = $1 AND user_id = $2);

-- name: CountMembershipsByGroupID :one
SELECT COUNT(*) FROM memberships WHERE group_id = $1;

-- name: DeleteMembershipsByGroupID :exec
DELETE FROM memberships WHERE group_id = $1;
