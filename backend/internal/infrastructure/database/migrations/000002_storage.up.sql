-- Storage Context Tables
-- Tables: folders, folder_paths, files, file_versions, archived_files, archived_file_versions, upload_sessions, upload_parts

-- Create enums
CREATE TYPE owner_type AS ENUM ('user', 'group');
CREATE TYPE file_status AS ENUM ('uploading', 'active', 'upload_failed');
CREATE TYPE upload_session_status AS ENUM ('pending', 'in_progress', 'completed', 'aborted', 'expired');

-- =====================================================
-- Folders table
-- =====================================================
CREATE TABLE folders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    parent_id UUID REFERENCES folders(id) ON DELETE CASCADE,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    owner_type owner_type NOT NULL DEFAULT 'user',
    depth INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_folders_parent_id ON folders(parent_id);
CREATE INDEX idx_folders_owner ON folders(owner_id, owner_type);

CREATE TRIGGER update_folders_updated_at
    BEFORE UPDATE ON folders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =====================================================
-- Folder paths table (closure table for hierarchy)
-- =====================================================
CREATE TABLE folder_paths (
    ancestor_id UUID NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
    descendant_id UUID NOT NULL REFERENCES folders(id) ON DELETE CASCADE,
    path_length INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (ancestor_id, descendant_id)
);

CREATE INDEX idx_folder_paths_ancestor ON folder_paths(ancestor_id);
CREATE INDEX idx_folder_paths_descendant ON folder_paths(descendant_id);

-- =====================================================
-- Files table
-- =====================================================
CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    owner_type owner_type NOT NULL DEFAULT 'user',
    folder_id UUID REFERENCES folders(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    storage_key TEXT NOT NULL,
    current_version INTEGER NOT NULL DEFAULT 1,
    status file_status NOT NULL DEFAULT 'uploading',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_files_folder_id ON files(folder_id);
CREATE INDEX idx_files_owner ON files(owner_id, owner_type);
CREATE INDEX idx_files_status ON files(status);
CREATE INDEX idx_files_storage_key ON files(storage_key);

CREATE TRIGGER update_files_updated_at
    BEFORE UPDATE ON files
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =====================================================
-- File versions table
-- =====================================================
CREATE TABLE file_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    minio_version_id VARCHAR(255),
    size BIGINT NOT NULL,
    checksum VARCHAR(64) NOT NULL,
    uploaded_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (file_id, version_number)
);

CREATE INDEX idx_file_versions_file_id ON file_versions(file_id);
CREATE INDEX idx_file_versions_minio_version ON file_versions(minio_version_id);

-- =====================================================
-- Archived files table (trash)
-- =====================================================
CREATE TABLE archived_files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    original_file_id UUID NOT NULL,
    original_folder_id UUID,
    original_path TEXT NOT NULL,
    name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    owner_type owner_type NOT NULL,
    storage_key TEXT NOT NULL,
    archived_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    archived_by UUID NOT NULL REFERENCES users(id),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_archived_files_owner ON archived_files(owner_id, owner_type);
CREATE INDEX idx_archived_files_original ON archived_files(original_file_id);
CREATE INDEX idx_archived_files_expires ON archived_files(expires_at);

-- =====================================================
-- Archived file versions table
-- =====================================================
CREATE TABLE archived_file_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    archived_file_id UUID NOT NULL REFERENCES archived_files(id) ON DELETE CASCADE,
    original_version_id UUID NOT NULL,
    version_number INTEGER NOT NULL,
    minio_version_id VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    checksum VARCHAR(64) NOT NULL,
    uploaded_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_archived_file_versions_file ON archived_file_versions(archived_file_id);

-- =====================================================
-- Upload sessions table
-- =====================================================
CREATE TABLE upload_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    owner_type owner_type NOT NULL DEFAULT 'user',
    folder_id UUID REFERENCES folders(id) ON DELETE SET NULL,
    file_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(255) NOT NULL,
    total_size BIGINT NOT NULL,
    storage_key TEXT NOT NULL,
    minio_upload_id VARCHAR(255),
    is_multipart BOOLEAN NOT NULL DEFAULT FALSE,
    total_parts INTEGER NOT NULL DEFAULT 1,
    uploaded_parts INTEGER NOT NULL DEFAULT 0,
    status upload_session_status NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_upload_sessions_file_id ON upload_sessions(file_id);
CREATE INDEX idx_upload_sessions_status ON upload_sessions(status);
CREATE INDEX idx_upload_sessions_storage_key ON upload_sessions(storage_key);

CREATE TRIGGER update_upload_sessions_updated_at
    BEFORE UPDATE ON upload_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =====================================================
-- Upload parts table
-- =====================================================
CREATE TABLE upload_parts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES upload_sessions(id) ON DELETE CASCADE,
    part_number INTEGER NOT NULL,
    size BIGINT NOT NULL,
    etag VARCHAR(255) NOT NULL,
    uploaded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (session_id, part_number)
);

CREATE INDEX idx_upload_parts_session ON upload_parts(session_id);
