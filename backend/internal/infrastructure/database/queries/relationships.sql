-- name: CreateRelationship :one
INSERT INTO relationships (
    id, subject_type, subject_id, relation, object_type, object_id, created_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: DeleteRelationship :exec
DELETE FROM relationships WHERE id = $1;

-- name: DeleteRelationshipByTuple :exec
DELETE FROM relationships
WHERE subject_type = $1 AND subject_id = $2 AND relation = $3 AND object_type = $4 AND object_id = $5;

-- name: RelationshipExists :one
SELECT EXISTS(
    SELECT 1 FROM relationships
    WHERE subject_type = $1 AND subject_id = $2 AND relation = $3 AND object_type = $4 AND object_id = $5
);

-- name: FindSubjectsByObject :many
SELECT subject_type, subject_id FROM relationships
WHERE object_type = $1 AND object_id = $2 AND relation = $3;

-- name: FindObjectsBySubject :many
SELECT object_id FROM relationships
WHERE subject_type = $1 AND subject_id = $2 AND relation = $3 AND object_type = $4;

-- name: ListRelationshipsByObject :many
SELECT * FROM relationships
WHERE object_type = $1 AND object_id = $2
ORDER BY created_at DESC;

-- name: ListRelationshipsBySubject :many
SELECT * FROM relationships
WHERE subject_type = $1 AND subject_id = $2
ORDER BY created_at DESC;

-- name: FindParentRelationship :one
SELECT subject_type, subject_id FROM relationships
WHERE object_type = $1 AND object_id = $2 AND relation = 'parent'
LIMIT 1;

-- name: DeleteRelationshipsByObject :exec
DELETE FROM relationships WHERE object_type = $1 AND object_id = $2;

-- name: DeleteRelationshipsBySubject :exec
DELETE FROM relationships WHERE subject_type = $1 AND subject_id = $2;
