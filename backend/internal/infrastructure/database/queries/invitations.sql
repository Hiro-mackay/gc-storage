-- name: CreateInvitation :one
INSERT INTO invitations (
    id, group_id, email, token, role, invited_by, expires_at, status, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetInvitationByID :one
SELECT * FROM invitations WHERE id = $1;

-- name: UpdateInvitation :one
UPDATE invitations SET
    status = COALESCE(sqlc.narg('status'), status)
WHERE id = $1
RETURNING *;

-- name: DeleteInvitation :exec
DELETE FROM invitations WHERE id = $1;

-- name: GetInvitationByToken :one
SELECT * FROM invitations WHERE token = $1;

-- name: ListPendingInvitationsByGroupID :many
SELECT * FROM invitations
WHERE group_id = $1 AND status = 'pending'
ORDER BY created_at DESC;

-- name: ListPendingInvitationsByEmail :many
SELECT * FROM invitations
WHERE email = $1 AND status = 'pending'
ORDER BY created_at DESC;

-- name: GetPendingInvitationByGroupAndEmail :one
SELECT * FROM invitations
WHERE group_id = $1 AND email = $2 AND status = 'pending';

-- name: DeleteInvitationsByGroupID :exec
DELETE FROM invitations WHERE group_id = $1;

-- name: ExpireOldInvitations :execrows
UPDATE invitations
SET status = 'expired'
WHERE status = 'pending' AND expires_at < NOW();
