-- Authorization Context Tables
-- Tables: permission_grants, relationships

-- =====================================================
-- Permission grants table
-- Role-based permission grants to users/groups on resources
-- =====================================================
CREATE TABLE permission_grants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_type VARCHAR(20) NOT NULL CHECK (resource_type IN ('file', 'folder')),
    resource_id UUID NOT NULL,
    grantee_type VARCHAR(20) NOT NULL CHECK (grantee_type IN ('user', 'group')),
    grantee_id UUID NOT NULL,
    role VARCHAR(30) NOT NULL CHECK (role IN ('viewer', 'contributor', 'content_manager', 'owner')),
    granted_by UUID NOT NULL REFERENCES users(id),
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(resource_type, resource_id, grantee_type, grantee_id, role)
);

CREATE INDEX idx_permission_grants_resource ON permission_grants(resource_type, resource_id);
CREATE INDEX idx_permission_grants_grantee ON permission_grants(grantee_type, grantee_id);
CREATE INDEX idx_permission_grants_role ON permission_grants(role);
CREATE INDEX idx_permission_grants_granted_by ON permission_grants(granted_by);

-- =====================================================
-- Relationships table (Zanzibar-style tuples)
-- For ownership and hierarchy relationships
-- =====================================================
CREATE TABLE relationships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subject_type VARCHAR(20) NOT NULL CHECK (subject_type IN ('user', 'group', 'file', 'folder')),
    subject_id UUID NOT NULL,
    relation VARCHAR(30) NOT NULL CHECK (relation IN ('owner', 'member', 'parent')),
    object_type VARCHAR(20) NOT NULL CHECK (object_type IN ('file', 'folder', 'group')),
    object_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(subject_type, subject_id, relation, object_type, object_id)
);

CREATE INDEX idx_relationships_subject ON relationships(subject_type, subject_id);
CREATE INDEX idx_relationships_object ON relationships(object_type, object_id);
CREATE INDEX idx_relationships_relation ON relationships(relation);
CREATE INDEX idx_relationships_object_relation ON relationships(object_type, object_id, relation);
