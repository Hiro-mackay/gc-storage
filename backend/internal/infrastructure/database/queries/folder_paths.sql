-- name: InsertFolderPath :exec
INSERT INTO folder_paths (ancestor_id, descendant_id, path_length, created_at)
VALUES ($1, $2, $3, NOW());

-- name: InsertFolderPathsBulk :copyfrom
INSERT INTO folder_paths (ancestor_id, descendant_id, path_length, created_at)
VALUES ($1, $2, $3, $4);

-- name: DeleteFolderPathsByDescendant :exec
DELETE FROM folder_paths WHERE descendant_id = $1;

-- name: GetAncestorIDs :many
SELECT ancestor_id FROM folder_paths
WHERE descendant_id = $1 AND path_length > 0
ORDER BY path_length ASC;

-- name: GetDescendantIDs :many
SELECT descendant_id FROM folder_paths
WHERE ancestor_id = $1 AND path_length > 0
ORDER BY path_length ASC;

-- name: GetAncestorPaths :many
-- Excludes self-reference (path_length = 0) to avoid duplicate when creating new folder
SELECT * FROM folder_paths
WHERE descendant_id = $1 AND path_length > 0
ORDER BY path_length ASC;

-- name: GetDescendantsWithDepth :many
SELECT descendant_id, path_length FROM folder_paths
WHERE ancestor_id = $1 AND path_length > 0;

-- name: DeleteSubtreePaths :exec
DELETE FROM folder_paths fp_outer
WHERE fp_outer.descendant_id IN (
    SELECT fp_inner.descendant_id FROM folder_paths fp_inner WHERE fp_inner.ancestor_id = $1
);

-- name: GetSelfAndDescendantIDs :many
SELECT descendant_id FROM folder_paths
WHERE ancestor_id = $1
ORDER BY path_length ASC;
