-- Migration rollback: Revert ownership model changes

-- =====================================================
-- Step 1: Drop new indexes
-- =====================================================

DROP INDEX IF EXISTS idx_folders_owner_id;
DROP INDEX IF EXISTS idx_folders_created_by;
DROP INDEX IF EXISTS idx_files_owner_id;
DROP INDEX IF EXISTS idx_files_created_by;
DROP INDEX IF EXISTS idx_archived_files_owner_id;
DROP INDEX IF EXISTS idx_archived_files_created_by;
DROP INDEX IF EXISTS idx_users_personal_folder;

-- =====================================================
-- Step 2: Recreate owner_type enum
-- =====================================================

CREATE TYPE owner_type AS ENUM ('user', 'group');

-- =====================================================
-- Step 3: Add owner_type columns back
-- =====================================================

ALTER TABLE folders ADD COLUMN owner_type owner_type NOT NULL DEFAULT 'user';
ALTER TABLE files ADD COLUMN owner_type owner_type NOT NULL DEFAULT 'user';
ALTER TABLE upload_sessions ADD COLUMN owner_type owner_type NOT NULL DEFAULT 'user';
ALTER TABLE archived_files ADD COLUMN owner_type owner_type NOT NULL DEFAULT 'user';

-- =====================================================
-- Step 4: Recreate owner indexes
-- =====================================================

CREATE INDEX idx_folders_owner ON folders(owner_id, owner_type);
CREATE INDEX idx_files_owner ON files(owner_id, owner_type);
CREATE INDEX idx_archived_files_owner ON archived_files(owner_id, owner_type);

-- =====================================================
-- Step 5: Make folder_id nullable again
-- =====================================================

ALTER TABLE files ALTER COLUMN folder_id DROP NOT NULL;
ALTER TABLE upload_sessions ALTER COLUMN folder_id DROP NOT NULL;
ALTER TABLE archived_files ALTER COLUMN original_folder_id DROP NOT NULL;

-- =====================================================
-- Step 6: Drop new columns
-- =====================================================

ALTER TABLE users DROP COLUMN personal_folder_id;
ALTER TABLE folders DROP COLUMN status;
ALTER TABLE folders DROP COLUMN created_by;
ALTER TABLE files DROP COLUMN created_by;
ALTER TABLE upload_sessions DROP COLUMN created_by;
ALTER TABLE archived_files DROP COLUMN created_by;
