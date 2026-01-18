-- name: CreateInvitation :one
INSERT INTO invitations (
    id, group_id, email, role, token, invited_by, status, expires_at, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetInvitationByID :one
SELECT * FROM invitations WHERE id = $1;

-- name: GetInvitationByToken :one
SELECT * FROM invitations WHERE token = $1;

-- name: GetPendingInvitation :one
SELECT * FROM invitations
WHERE group_id = $1 AND email = $2 AND status = 'pending';

-- name: AcceptInvitation :exec
UPDATE invitations SET status = 'accepted', accepted_at = NOW() WHERE id = $1;

-- name: RevokeInvitation :exec
UPDATE invitations SET status = 'revoked' WHERE id = $1;

-- name: ExpireInvitation :exec
UPDATE invitations SET status = 'expired' WHERE id = $1;

-- name: ListInvitationsByGroupID :many
SELECT * FROM invitations
WHERE group_id = $1
ORDER BY created_at DESC;

-- name: ListPendingInvitationsByEmail :many
SELECT * FROM invitations
WHERE email = $1 AND status = 'pending' AND expires_at > NOW()
ORDER BY created_at DESC;

-- name: DeleteExpiredInvitations :exec
DELETE FROM invitations WHERE expires_at < NOW() AND status = 'pending';

-- name: CountPendingInvitationsByGroupID :one
SELECT COUNT(*) FROM invitations WHERE group_id = $1 AND status = 'pending';
