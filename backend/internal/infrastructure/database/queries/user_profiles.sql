-- name: CreateUserProfile :one
INSERT INTO user_profiles (
    user_id, display_name, avatar_url, bio, locale, timezone, settings, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetUserProfileByUserID :one
SELECT * FROM user_profiles WHERE user_id = $1;

-- name: UpdateUserProfile :one
UPDATE user_profiles SET
    display_name = COALESCE(sqlc.narg('display_name'), display_name),
    avatar_url = COALESCE(sqlc.narg('avatar_url'), avatar_url),
    bio = COALESCE(sqlc.narg('bio'), bio),
    locale = COALESCE(sqlc.narg('locale'), locale),
    timezone = COALESCE(sqlc.narg('timezone'), timezone),
    settings = COALESCE(sqlc.narg('settings'), settings),
    updated_at = NOW()
WHERE user_id = $1
RETURNING *;

-- name: UpsertUserProfile :one
INSERT INTO user_profiles (
    user_id, display_name, avatar_url, bio, locale, timezone, settings, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, NOW()
)
ON CONFLICT (user_id) DO UPDATE SET
    display_name = COALESCE(EXCLUDED.display_name, user_profiles.display_name),
    avatar_url = COALESCE(EXCLUDED.avatar_url, user_profiles.avatar_url),
    bio = COALESCE(EXCLUDED.bio, user_profiles.bio),
    locale = COALESCE(EXCLUDED.locale, user_profiles.locale),
    timezone = COALESCE(EXCLUDED.timezone, user_profiles.timezone),
    settings = COALESCE(EXCLUDED.settings, user_profiles.settings),
    updated_at = NOW()
RETURNING *;

-- name: DeleteUserProfile :exec
DELETE FROM user_profiles WHERE user_id = $1;
