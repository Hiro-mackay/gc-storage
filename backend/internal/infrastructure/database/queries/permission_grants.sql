-- name: CreatePermissionGrant :one
INSERT INTO permission_grants (
    id, resource_type, resource_id, grantee_type, grantee_id, permission, granted_by, created_at, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetPermissionGrant :one
SELECT * FROM permission_grants
WHERE resource_type = $1 AND resource_id = $2 AND grantee_type = $3 AND grantee_id = $4 AND permission = $5;

-- name: GetPermissionGrantByID :one
SELECT * FROM permission_grants WHERE id = $1;

-- name: ListPermissionGrantsByResource :many
SELECT * FROM permission_grants
WHERE resource_type = $1 AND resource_id = $2
ORDER BY created_at ASC;

-- name: ListPermissionGrantsByGrantee :many
SELECT * FROM permission_grants
WHERE grantee_type = $1 AND grantee_id = $2
ORDER BY created_at ASC;

-- name: CheckPermission :one
SELECT EXISTS(
    SELECT 1 FROM permission_grants
    WHERE resource_type = $1
      AND resource_id = $2
      AND grantee_type = $3
      AND grantee_id = $4
      AND permission = $5
      AND (expires_at IS NULL OR expires_at > NOW())
);

-- name: ListUserPermissions :many
SELECT * FROM permission_grants
WHERE grantee_type = 'user' AND grantee_id = $1
AND (expires_at IS NULL OR expires_at > NOW());

-- name: DeletePermissionGrant :exec
DELETE FROM permission_grants WHERE id = $1;

-- name: DeletePermissionGrantsByResource :exec
DELETE FROM permission_grants
WHERE resource_type = $1 AND resource_id = $2;

-- name: DeletePermissionGrantsByGrantee :exec
DELETE FROM permission_grants
WHERE grantee_type = $1 AND grantee_id = $2;

-- name: DeleteExpiredPermissionGrants :exec
DELETE FROM permission_grants WHERE expires_at IS NOT NULL AND expires_at < NOW();
