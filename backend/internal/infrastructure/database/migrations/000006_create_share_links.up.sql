-- Sharing Context Tables
-- Tables: share_links, share_link_accesses

-- =====================================================
-- Share links table
-- =====================================================
CREATE TABLE share_links (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    token VARCHAR(64) NOT NULL UNIQUE,
    resource_type VARCHAR(20) NOT NULL CHECK (resource_type IN ('file', 'folder')),
    resource_id UUID NOT NULL,
    created_by UUID NOT NULL REFERENCES users(id),
    permission VARCHAR(20) NOT NULL DEFAULT 'read' CHECK (permission IN ('read', 'write')),
    password_hash VARCHAR(255),
    expires_at TIMESTAMPTZ,
    max_access_count INT,
    access_count INT NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked', 'expired')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_share_links_token ON share_links(token);
CREATE INDEX idx_share_links_resource ON share_links(resource_type, resource_id);
CREATE INDEX idx_share_links_created_by ON share_links(created_by);
CREATE INDEX idx_share_links_status ON share_links(status);
CREATE INDEX idx_share_links_expires_at ON share_links(expires_at);

CREATE TRIGGER update_share_links_updated_at
    BEFORE UPDATE ON share_links
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =====================================================
-- Share link accesses table
-- =====================================================
CREATE TABLE share_link_accesses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    share_link_id UUID NOT NULL REFERENCES share_links(id) ON DELETE CASCADE,
    accessed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ip_address VARCHAR(45),
    user_agent TEXT,
    user_id UUID REFERENCES users(id),
    action VARCHAR(20) NOT NULL CHECK (action IN ('view', 'download', 'upload'))
);

CREATE INDEX idx_share_link_accesses_link_id ON share_link_accesses(share_link_id);
CREATE INDEX idx_share_link_accesses_accessed_at ON share_link_accesses(accessed_at);
CREATE INDEX idx_share_link_accesses_user_id ON share_link_accesses(user_id);
