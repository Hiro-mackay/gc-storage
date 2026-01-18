-- name: CreateOAuthAccount :one
INSERT INTO oauth_accounts (
    id, user_id, provider, provider_user_id, access_token, refresh_token, expires_at, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetOAuthAccountByProviderAndUserID :one
SELECT * FROM oauth_accounts
WHERE provider = $1 AND provider_user_id = $2;

-- name: GetOAuthAccountsByUserID :many
SELECT * FROM oauth_accounts WHERE user_id = $1;

-- name: UpdateOAuthTokens :exec
UPDATE oauth_accounts SET
    access_token = $2,
    refresh_token = $3,
    expires_at = $4,
    updated_at = NOW()
WHERE id = $1;

-- name: DeleteOAuthAccount :exec
DELETE FROM oauth_accounts WHERE id = $1;

-- name: DeleteOAuthAccountByUserAndProvider :exec
DELETE FROM oauth_accounts WHERE user_id = $1 AND provider = $2;
