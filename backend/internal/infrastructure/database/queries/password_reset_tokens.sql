-- name: CreatePasswordResetToken :one
INSERT INTO password_reset_tokens (id, user_id, token, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: GetPasswordResetTokenByToken :one
SELECT * FROM password_reset_tokens WHERE token = $1;

-- name: MarkPasswordResetTokenAsUsed :exec
UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1;

-- name: DeletePasswordResetToken :exec
DELETE FROM password_reset_tokens WHERE id = $1;

-- name: DeletePasswordResetTokensByUserID :exec
DELETE FROM password_reset_tokens WHERE user_id = $1;

-- name: DeleteExpiredPasswordResetTokens :exec
DELETE FROM password_reset_tokens WHERE expires_at < NOW();
