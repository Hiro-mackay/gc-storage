-- Revert display_name column to 100 characters
ALTER TABLE users ALTER COLUMN display_name TYPE VARCHAR(100);
