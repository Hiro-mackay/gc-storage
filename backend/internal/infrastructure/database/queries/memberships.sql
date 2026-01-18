-- name: CreateMembership :one
INSERT INTO memberships (
    id, group_id, user_id, role, joined_at
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetMembership :one
SELECT * FROM memberships
WHERE group_id = $1 AND user_id = $2;

-- name: GetMembershipByID :one
SELECT * FROM memberships WHERE id = $1;

-- name: UpdateMembershipRole :exec
UPDATE memberships SET role = $3
WHERE group_id = $1 AND user_id = $2;

-- name: DeleteMembership :exec
DELETE FROM memberships WHERE group_id = $1 AND user_id = $2;

-- name: ListMembershipsByGroupID :many
SELECT * FROM memberships
WHERE group_id = $1
ORDER BY joined_at ASC;

-- name: ListMembershipsByUserID :many
SELECT * FROM memberships WHERE user_id = $1;

-- name: CountMembershipsByGroupID :one
SELECT COUNT(*) FROM memberships WHERE group_id = $1;

-- name: MembershipExists :one
SELECT EXISTS(SELECT 1 FROM memberships WHERE group_id = $1 AND user_id = $2);
