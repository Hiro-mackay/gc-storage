-- Create user_profiles table
CREATE TABLE user_profiles (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    display_name VARCHAR(100),
    avatar_url TEXT,
    bio TEXT CHECK (char_length(bio) <= 500),
    locale VARCHAR(10) NOT NULL DEFAULT 'ja',
    timezone VARCHAR(50) NOT NULL DEFAULT 'Asia/Tokyo',
    settings JSONB NOT NULL DEFAULT '{}',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create index for user_id lookup (primary key already indexed)
CREATE INDEX idx_user_profiles_locale ON user_profiles(locale);

-- Apply trigger to user_profiles table
CREATE TRIGGER update_user_profiles_updated_at
    BEFORE UPDATE ON user_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Insert default profiles for existing users
INSERT INTO user_profiles (user_id, display_name, avatar_url)
SELECT id, display_name, avatar_url FROM users;
