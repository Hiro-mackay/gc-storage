-- Restore avatar_url column to users table
ALTER TABLE users ADD COLUMN avatar_url TEXT;

-- Copy avatar_url data back from user_profiles
UPDATE users u
SET avatar_url = p.avatar_url
FROM user_profiles p
WHERE u.id = p.user_id
  AND p.avatar_url IS NOT NULL;
