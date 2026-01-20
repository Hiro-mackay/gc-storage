-- name: CreateUserProfile :one
INSERT INTO user_profiles (
    id, user_id, avatar_url, bio, timezone, locale, theme, notification_preferences, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetUserProfileByUserID :one
SELECT * FROM user_profiles WHERE user_id = $1;

-- name: UpdateUserProfile :one
UPDATE user_profiles SET
    avatar_url = COALESCE(sqlc.narg('avatar_url'), avatar_url),
    bio = COALESCE(sqlc.narg('bio'), bio),
    locale = COALESCE(sqlc.narg('locale'), locale),
    timezone = COALESCE(sqlc.narg('timezone'), timezone),
    theme = COALESCE(sqlc.narg('theme'), theme),
    notification_preferences = COALESCE(sqlc.narg('notification_preferences'), notification_preferences),
    updated_at = NOW()
WHERE user_id = $1
RETURNING *;

-- name: UpsertUserProfile :one
INSERT INTO user_profiles (
    id, user_id, avatar_url, bio, locale, timezone, theme, notification_preferences, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW()
)
ON CONFLICT (user_id) DO UPDATE SET
    avatar_url = COALESCE(EXCLUDED.avatar_url, user_profiles.avatar_url),
    bio = COALESCE(EXCLUDED.bio, user_profiles.bio),
    locale = COALESCE(EXCLUDED.locale, user_profiles.locale),
    timezone = COALESCE(EXCLUDED.timezone, user_profiles.timezone),
    theme = COALESCE(EXCLUDED.theme, user_profiles.theme),
    notification_preferences = COALESCE(EXCLUDED.notification_preferences, user_profiles.notification_preferences),
    updated_at = NOW()
RETURNING *;

-- name: DeleteUserProfile :exec
DELETE FROM user_profiles WHERE user_id = $1;
