-- Create permission_grants table (PBAC - Policy-Based Access Control)
CREATE TABLE permission_grants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_type VARCHAR(50) NOT NULL CHECK (resource_type IN ('file', 'folder')),
    resource_id UUID NOT NULL,
    grantee_type VARCHAR(50) NOT NULL CHECK (grantee_type IN ('user', 'group')),
    grantee_id UUID NOT NULL,
    permission VARCHAR(20) NOT NULL CHECK (permission IN ('read', 'write', 'delete', 'share', 'owner')),
    granted_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    UNIQUE (resource_type, resource_id, grantee_type, grantee_id, permission)
);

-- Create relationships table (ReBAC - Relationship-Based Access Control / Zanzibar-style)
CREATE TABLE relationships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    object_type VARCHAR(50) NOT NULL,
    object_id UUID NOT NULL,
    relation VARCHAR(50) NOT NULL,
    subject_type VARCHAR(50) NOT NULL,
    subject_id UUID NOT NULL,
    subject_relation VARCHAR(50),  -- For nested relations like "group#member"
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (object_type, object_id, relation, subject_type, subject_id, subject_relation)
);

-- Create indexes
CREATE INDEX idx_permission_grants_resource ON permission_grants(resource_type, resource_id);
CREATE INDEX idx_permission_grants_grantee ON permission_grants(grantee_type, grantee_id);
CREATE INDEX idx_permission_grants_permission ON permission_grants(permission);

CREATE INDEX idx_relationships_object ON relationships(object_type, object_id);
CREATE INDEX idx_relationships_subject ON relationships(subject_type, subject_id);
CREATE INDEX idx_relationships_relation ON relationships(relation);
CREATE INDEX idx_relationships_lookup ON relationships(object_type, object_id, relation);

-- Composite index for permission checks
CREATE INDEX idx_permission_grants_check ON permission_grants(resource_type, resource_id, grantee_type, grantee_id, permission);
