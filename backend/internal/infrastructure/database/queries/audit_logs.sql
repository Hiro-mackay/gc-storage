-- name: CreateAuditLog :one
INSERT INTO audit_logs (user_id, action, resource_type, resource_id, details, ip_address, user_agent, request_id)
VALUES (@user_id, @action, @resource_type, @resource_id, @details, @ip_address, @user_agent, @request_id)
RETURNING *;

-- name: ListAuditLogsByUserID :many
SELECT * FROM audit_logs
WHERE user_id = @user_id
ORDER BY created_at DESC
LIMIT @limit_val OFFSET @offset_val;

-- name: ListAuditLogsByResource :many
SELECT * FROM audit_logs
WHERE resource_type = @resource_type AND resource_id = @resource_id
ORDER BY created_at DESC
LIMIT @limit_val OFFSET @offset_val;

-- name: CountAuditLogsByUserID :one
SELECT COUNT(*) FROM audit_logs
WHERE user_id = @user_id;
