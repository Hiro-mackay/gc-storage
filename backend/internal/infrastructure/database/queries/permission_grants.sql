-- name: CreatePermissionGrant :one
INSERT INTO permission_grants (
    id, resource_type, resource_id, grantee_type, grantee_id, role, granted_by, granted_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetPermissionGrantByID :one
SELECT * FROM permission_grants WHERE id = $1;

-- name: DeletePermissionGrant :exec
DELETE FROM permission_grants WHERE id = $1;

-- name: ListPermissionGrantsByResource :many
SELECT * FROM permission_grants
WHERE resource_type = $1 AND resource_id = $2
ORDER BY granted_at DESC;

-- name: ListPermissionGrantsByResourceAndGrantee :many
SELECT * FROM permission_grants
WHERE resource_type = $1 AND resource_id = $2 AND grantee_type = $3 AND grantee_id = $4
ORDER BY granted_at DESC;

-- name: GetPermissionGrantByResourceGranteeAndRole :one
SELECT * FROM permission_grants
WHERE resource_type = $1 AND resource_id = $2 AND grantee_type = $3 AND grantee_id = $4 AND role = $5;

-- name: ListPermissionGrantsByGrantee :many
SELECT * FROM permission_grants
WHERE grantee_type = $1 AND grantee_id = $2
ORDER BY granted_at DESC;

-- name: DeletePermissionGrantsByResource :exec
DELETE FROM permission_grants WHERE resource_type = $1 AND resource_id = $2;

-- name: DeletePermissionGrantsByGrantee :exec
DELETE FROM permission_grants WHERE grantee_type = $1 AND grantee_id = $2;

-- name: DeletePermissionGrantsByResourceAndGrantee :exec
DELETE FROM permission_grants
WHERE resource_type = $1 AND resource_id = $2 AND grantee_type = $3 AND grantee_id = $4;
