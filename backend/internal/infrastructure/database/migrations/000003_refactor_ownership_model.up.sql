-- Migration: Refactor ownership model
-- Changes:
-- 1. Remove owner_type from folders, files, upload_sessions, archived_files
-- 2. Add created_by to folders, files, upload_sessions, archived_files
-- 3. Make folder_id NOT NULL in files, upload_sessions
-- 4. Make original_folder_id NOT NULL in archived_files
-- 5. Add status to folders
-- 6. Add personal_folder_id to users

-- =====================================================
-- Step 1: Add new columns first
-- =====================================================

-- Add created_by to folders
ALTER TABLE folders ADD COLUMN created_by UUID NOT NULL REFERENCES users(id) DEFAULT '00000000-0000-0000-0000-000000000000';

-- Add status to folders
ALTER TABLE folders ADD COLUMN status VARCHAR(50) NOT NULL DEFAULT 'active';

-- Add created_by to files
ALTER TABLE files ADD COLUMN created_by UUID NOT NULL REFERENCES users(id) DEFAULT '00000000-0000-0000-0000-000000000000';

-- Add created_by to upload_sessions
ALTER TABLE upload_sessions ADD COLUMN created_by UUID NOT NULL REFERENCES users(id) DEFAULT '00000000-0000-0000-0000-000000000000';

-- Add created_by to archived_files
ALTER TABLE archived_files ADD COLUMN created_by UUID NOT NULL REFERENCES users(id) DEFAULT '00000000-0000-0000-0000-000000000000';

-- Add personal_folder_id to users
ALTER TABLE users ADD COLUMN personal_folder_id UUID REFERENCES folders(id);

-- =====================================================
-- Step 2: Migrate data - set created_by = owner_id for existing records
-- =====================================================

UPDATE folders SET created_by = owner_id WHERE created_by = '00000000-0000-0000-0000-000000000000';
UPDATE files SET created_by = owner_id WHERE created_by = '00000000-0000-0000-0000-000000000000';
UPDATE upload_sessions SET created_by = owner_id WHERE created_by = '00000000-0000-0000-0000-000000000000';
UPDATE archived_files SET created_by = owner_id WHERE created_by = '00000000-0000-0000-0000-000000000000';

-- =====================================================
-- Step 3: Remove default constraints
-- =====================================================

ALTER TABLE folders ALTER COLUMN created_by DROP DEFAULT;
ALTER TABLE files ALTER COLUMN created_by DROP DEFAULT;
ALTER TABLE upload_sessions ALTER COLUMN created_by DROP DEFAULT;
ALTER TABLE archived_files ALTER COLUMN created_by DROP DEFAULT;

-- =====================================================
-- Step 4: Make folder_id NOT NULL where required
-- Note: This requires existing data to have valid folder_id values
-- For files without folder_id, we need to handle them first
-- =====================================================

-- For files: Create a default folder for orphan files or fail if there are orphans
-- Check and fail if any files have NULL folder_id
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM files WHERE folder_id IS NULL) THEN
        RAISE EXCEPTION 'Cannot migrate: files with NULL folder_id exist. Please assign them to folders first.';
    END IF;
END $$;

-- Make folder_id NOT NULL on files
ALTER TABLE files ALTER COLUMN folder_id SET NOT NULL;

-- For upload_sessions: Check and fail if any have NULL folder_id
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM upload_sessions WHERE folder_id IS NULL) THEN
        RAISE EXCEPTION 'Cannot migrate: upload_sessions with NULL folder_id exist. Please clean them up first.';
    END IF;
END $$;

-- Make folder_id NOT NULL on upload_sessions
ALTER TABLE upload_sessions ALTER COLUMN folder_id SET NOT NULL;

-- For archived_files: Check and fail if any have NULL original_folder_id
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM archived_files WHERE original_folder_id IS NULL) THEN
        RAISE EXCEPTION 'Cannot migrate: archived_files with NULL original_folder_id exist. Please clean them up first.';
    END IF;
END $$;

-- Make original_folder_id NOT NULL on archived_files
ALTER TABLE archived_files ALTER COLUMN original_folder_id SET NOT NULL;

-- =====================================================
-- Step 5: Drop owner_type columns
-- =====================================================

-- Drop owner_type related indexes first
DROP INDEX IF EXISTS idx_folders_owner;
DROP INDEX IF EXISTS idx_files_owner;
DROP INDEX IF EXISTS idx_archived_files_owner;

-- Drop owner_type columns
ALTER TABLE folders DROP COLUMN owner_type;
ALTER TABLE files DROP COLUMN owner_type;
ALTER TABLE upload_sessions DROP COLUMN owner_type;
ALTER TABLE archived_files DROP COLUMN owner_type;

-- =====================================================
-- Step 6: Drop owner_type enum type
-- =====================================================

DROP TYPE IF EXISTS owner_type;

-- =====================================================
-- Step 7: Create new indexes
-- =====================================================

CREATE INDEX idx_folders_owner_id ON folders(owner_id);
CREATE INDEX idx_folders_created_by ON folders(created_by);
CREATE INDEX idx_files_owner_id ON files(owner_id);
CREATE INDEX idx_files_created_by ON files(created_by);
CREATE INDEX idx_archived_files_owner_id ON archived_files(owner_id);
CREATE INDEX idx_archived_files_created_by ON archived_files(created_by);
CREATE INDEX idx_users_personal_folder ON users(personal_folder_id);
