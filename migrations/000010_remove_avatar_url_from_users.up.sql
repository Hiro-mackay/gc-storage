-- Remove avatar_url from users table (data is now stored in user_profiles only)
-- First, ensure all avatar_url data is migrated to user_profiles
UPDATE user_profiles p
SET avatar_url = u.avatar_url
FROM users u
WHERE p.user_id = u.id
  AND p.avatar_url IS NULL
  AND u.avatar_url IS NOT NULL;

-- Drop avatar_url column from users
ALTER TABLE users DROP COLUMN avatar_url;
