-- name: CreateRelationship :one
INSERT INTO relationships (
    id, object_type, object_id, relation, subject_type, subject_id, subject_relation, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetRelationship :one
SELECT * FROM relationships
WHERE object_type = $1 AND object_id = $2 AND relation = $3 AND subject_type = $4 AND subject_id = $5;

-- name: GetRelationshipByID :one
SELECT * FROM relationships WHERE id = $1;

-- name: CheckRelationship :one
SELECT EXISTS(
    SELECT 1 FROM relationships
    WHERE object_type = $1
      AND object_id = $2
      AND relation = $3
      AND subject_type = $4
      AND subject_id = $5
);

-- name: ListRelationshipsByObject :many
SELECT * FROM relationships
WHERE object_type = $1 AND object_id = $2
ORDER BY created_at ASC;

-- name: ListRelationshipsByObjectAndRelation :many
SELECT * FROM relationships
WHERE object_type = $1 AND object_id = $2 AND relation = $3
ORDER BY created_at ASC;

-- name: ListRelationshipsBySubject :many
SELECT * FROM relationships
WHERE subject_type = $1 AND subject_id = $2
ORDER BY created_at ASC;

-- name: DeleteRelationship :exec
DELETE FROM relationships WHERE id = $1;

-- name: DeleteRelationshipByTuple :exec
DELETE FROM relationships
WHERE object_type = $1 AND object_id = $2 AND relation = $3 AND subject_type = $4 AND subject_id = $5;

-- name: DeleteRelationshipsByObject :exec
DELETE FROM relationships WHERE object_type = $1 AND object_id = $2;

-- name: DeleteRelationshipsBySubject :exec
DELETE FROM relationships WHERE subject_type = $1 AND subject_id = $2;

-- name: GetParentRelation :one
SELECT * FROM relationships
WHERE object_type = 'folder' AND object_id = $1 AND relation = 'parent'
LIMIT 1;

-- name: ListChildRelations :many
SELECT * FROM relationships
WHERE relation = 'parent' AND subject_type = 'folder' AND subject_id = $1;
