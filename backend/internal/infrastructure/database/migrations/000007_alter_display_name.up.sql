-- Alter display_name column to support up to 255 characters
ALTER TABLE users ALTER COLUMN display_name TYPE VARCHAR(255);
