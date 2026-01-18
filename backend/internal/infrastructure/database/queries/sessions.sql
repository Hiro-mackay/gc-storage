-- name: CreateSession :one
INSERT INTO sessions (
    id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetSessionByID :one
SELECT * FROM sessions WHERE id = $1 AND revoked_at IS NULL;

-- name: GetSessionByRefreshTokenHash :one
SELECT * FROM sessions WHERE refresh_token_hash = $1 AND revoked_at IS NULL;

-- name: ListSessionsByUserID :many
SELECT * FROM sessions
WHERE user_id = $1 AND revoked_at IS NULL
ORDER BY created_at DESC;

-- name: RevokeSession :exec
UPDATE sessions SET revoked_at = NOW() WHERE id = $1;

-- name: RevokeAllUserSessions :exec
UPDATE sessions SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < NOW() OR revoked_at IS NOT NULL;

-- name: CountActiveSessionsByUserID :one
SELECT COUNT(*) FROM sessions
WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW();

-- name: GetOldestActiveSession :one
SELECT * FROM sessions
WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
ORDER BY created_at ASC
LIMIT 1;
