-- name: CreateEmailVerificationToken :one
INSERT INTO email_verification_tokens (id, user_id, token, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5) RETURNING *;

-- name: GetEmailVerificationTokenByToken :one
SELECT * FROM email_verification_tokens WHERE token = $1;

-- name: GetEmailVerificationTokenByUserID :one
SELECT * FROM email_verification_tokens WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1;

-- name: DeleteEmailVerificationToken :exec
DELETE FROM email_verification_tokens WHERE id = $1;

-- name: DeleteEmailVerificationTokensByUserID :exec
DELETE FROM email_verification_tokens WHERE user_id = $1;

-- name: DeleteExpiredEmailVerificationTokens :exec
DELETE FROM email_verification_tokens WHERE expires_at < NOW();
