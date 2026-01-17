-- Create files table
CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    folder_id UUID REFERENCES folders(id) ON DELETE CASCADE,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mime_type VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    storage_key TEXT NOT NULL,  -- MinIO object key
    current_version INTEGER NOT NULL DEFAULT 1,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'trashed', 'deleted')),
    checksum VARCHAR(64),  -- SHA-256 hash
    trashed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create file_versions table
CREATE TABLE file_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    size BIGINT NOT NULL,
    storage_key TEXT NOT NULL,
    checksum VARCHAR(64),
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (file_id, version_number)
);

-- Create upload_sessions table
CREATE TABLE upload_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    upload_id VARCHAR(255),  -- MinIO multipart upload ID
    status VARCHAR(20) NOT NULL DEFAULT 'initiated' CHECK (status IN ('initiated', 'uploading', 'completing', 'completed', 'failed', 'aborted')),
    total_parts INTEGER,
    completed_parts INTEGER DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Create indexes
CREATE INDEX idx_files_folder_id ON files(folder_id);
CREATE INDEX idx_files_owner_id ON files(owner_id);
CREATE INDEX idx_files_status ON files(status);
CREATE INDEX idx_files_trashed_at ON files(trashed_at);
CREATE INDEX idx_files_storage_key ON files(storage_key);

CREATE INDEX idx_file_versions_file_id ON file_versions(file_id);
CREATE INDEX idx_file_versions_version ON file_versions(file_id, version_number);

CREATE INDEX idx_upload_sessions_file_id ON upload_sessions(file_id);
CREATE INDEX idx_upload_sessions_status ON upload_sessions(status);
CREATE INDEX idx_upload_sessions_expires_at ON upload_sessions(expires_at);

-- Unique constraint: no duplicate names in same folder for same owner
CREATE UNIQUE INDEX idx_files_unique_name ON files(folder_id, owner_id, name)
    WHERE status != 'deleted';

-- Apply updated_at trigger to files table
CREATE TRIGGER update_files_updated_at
    BEFORE UPDATE ON files
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
