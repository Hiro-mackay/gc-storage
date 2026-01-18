-- name: CreateUser :one
INSERT INTO users (
    id, email, password_hash, display_name, avatar_url, status, email_verified_at, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpdateUser :one
UPDATE users SET
    display_name = COALESCE(sqlc.narg('display_name'), display_name),
    avatar_url = COALESCE(sqlc.narg('avatar_url'), avatar_url),
    password_hash = COALESCE(sqlc.narg('password_hash'), password_hash),
    status = COALESCE(sqlc.narg('status'), status),
    email_verified_at = COALESCE(sqlc.narg('email_verified_at'), email_verified_at),
    last_login_at = COALESCE(sqlc.narg('last_login_at'), last_login_at),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateUserLastLogin :exec
UPDATE users SET last_login_at = NOW() WHERE id = $1;

-- name: VerifyUserEmail :exec
UPDATE users SET email_verified_at = NOW() WHERE id = $1;

-- name: UpdateUserStatus :exec
UPDATE users SET status = $2, updated_at = NOW() WHERE id = $1;

-- name: DeleteUser :exec
UPDATE users SET status = 'deleted', updated_at = NOW() WHERE id = $1;

-- name: UserExistsByEmail :one
SELECT EXISTS(SELECT 1 FROM users WHERE email = $1);

-- name: ListUsers :many
SELECT * FROM users
WHERE status != 'deleted'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountUsers :one
SELECT COUNT(*) FROM users WHERE status != 'deleted';
