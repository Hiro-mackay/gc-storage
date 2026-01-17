-- Create folders table
CREATE TABLE folders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    parent_id UUID REFERENCES folders(id) ON DELETE CASCADE,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    path TEXT NOT NULL,  -- Materialized path (e.g., "/uuid1/uuid2/uuid3")
    depth INTEGER NOT NULL DEFAULT 0,
    is_root BOOLEAN NOT NULL DEFAULT FALSE,
    trashed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_folders_parent_id ON folders(parent_id);
CREATE INDEX idx_folders_owner_id ON folders(owner_id);
CREATE INDEX idx_folders_path ON folders(path);
CREATE INDEX idx_folders_trashed_at ON folders(trashed_at);

-- Unique constraint: no duplicate names in same parent for same owner
CREATE UNIQUE INDEX idx_folders_unique_name ON folders(parent_id, owner_id, name)
    WHERE trashed_at IS NULL;

-- Root folder unique constraint: one root folder per user
CREATE UNIQUE INDEX idx_folders_unique_root ON folders(owner_id)
    WHERE is_root = TRUE;

-- Apply updated_at trigger to folders table
CREATE TRIGGER update_folders_updated_at
    BEFORE UPDATE ON folders
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
