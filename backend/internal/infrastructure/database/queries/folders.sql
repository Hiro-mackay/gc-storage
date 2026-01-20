-- name: CreateFolder :one
INSERT INTO folders (
    id, name, parent_id, owner_id, owner_type, depth, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetFolderByID :one
SELECT * FROM folders WHERE id = $1;

-- name: UpdateFolder :one
UPDATE folders SET
    name = COALESCE(sqlc.narg('name'), name),
    parent_id = COALESCE(sqlc.narg('parent_id'), parent_id),
    depth = COALESCE(sqlc.narg('depth'), depth),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateFolderDepth :exec
UPDATE folders SET depth = $2, updated_at = NOW() WHERE id = $1;

-- name: DeleteFolder :exec
DELETE FROM folders WHERE id = $1;

-- name: DeleteFoldersBulk :exec
DELETE FROM folders WHERE id = ANY($1::uuid[]);

-- name: ListFoldersByParentID :many
SELECT * FROM folders
WHERE parent_id = $1 AND owner_id = $2 AND owner_type = $3
ORDER BY name ASC;

-- name: ListRootFoldersByOwner :many
SELECT * FROM folders
WHERE parent_id IS NULL AND owner_id = $1 AND owner_type = $2
ORDER BY name ASC;

-- name: ListFoldersByOwner :many
SELECT * FROM folders
WHERE owner_id = $1 AND owner_type = $2
ORDER BY created_at DESC;

-- name: FolderExistsByNameAndParent :one
SELECT EXISTS(
    SELECT 1 FROM folders
    WHERE parent_id = $1 AND owner_id = $2 AND owner_type = $3 AND name = $4
);

-- name: FolderExistsByNameAtRoot :one
SELECT EXISTS(
    SELECT 1 FROM folders
    WHERE parent_id IS NULL AND owner_id = $1 AND owner_type = $2 AND name = $3
);

-- name: FolderExistsByID :one
SELECT EXISTS(SELECT 1 FROM folders WHERE id = $1);

-- name: BulkUpdateFolderDepth :exec
UPDATE folders SET depth = upd.depth, updated_at = NOW()
FROM (SELECT unnest($1::uuid[]) as id, unnest($2::int[]) as depth) as upd
WHERE folders.id = upd.id;
