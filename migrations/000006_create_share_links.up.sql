-- Create share_links table
CREATE TABLE share_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_type VARCHAR(50) NOT NULL CHECK (resource_type IN ('file', 'folder')),
    resource_id UUID NOT NULL,
    token VARCHAR(64) NOT NULL UNIQUE,
    permission VARCHAR(20) NOT NULL DEFAULT 'read' CHECK (permission IN ('read', 'write')),
    password_hash VARCHAR(255),
    max_downloads INTEGER,
    download_count INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create share_link_accesses table (access log)
CREATE TABLE share_link_accesses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    share_link_id UUID NOT NULL REFERENCES share_links(id) ON DELETE CASCADE,
    accessed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ip_address INET,
    user_agent TEXT,
    action VARCHAR(20) NOT NULL CHECK (action IN ('view', 'download')),
    accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_share_links_resource ON share_links(resource_type, resource_id);
CREATE INDEX idx_share_links_token ON share_links(token);
CREATE INDEX idx_share_links_created_by ON share_links(created_by);
CREATE INDEX idx_share_links_is_active ON share_links(is_active);
CREATE INDEX idx_share_links_expires_at ON share_links(expires_at);

CREATE INDEX idx_share_link_accesses_link_id ON share_link_accesses(share_link_id);
CREATE INDEX idx_share_link_accesses_accessed_at ON share_link_accesses(accessed_at);

-- Apply updated_at trigger to share_links table
CREATE TRIGGER update_share_links_updated_at
    BEFORE UPDATE ON share_links
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
