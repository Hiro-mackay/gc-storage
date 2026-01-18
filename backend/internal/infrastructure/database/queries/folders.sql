-- name: CreateFolder :one
INSERT INTO folders (
    id, name, parent_id, owner_id, path, depth, is_root, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
) RETURNING *;

-- name: GetFolderByID :one
SELECT * FROM folders WHERE id = $1;

-- name: GetRootFolderByOwnerID :one
SELECT * FROM folders WHERE owner_id = $1 AND is_root = TRUE;

-- name: UpdateFolder :one
UPDATE folders SET
    name = COALESCE(sqlc.narg('name'), name),
    parent_id = COALESCE(sqlc.narg('parent_id'), parent_id),
    path = COALESCE(sqlc.narg('path'), path),
    depth = COALESCE(sqlc.narg('depth'), depth),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: TrashFolder :exec
UPDATE folders SET trashed_at = NOW(), updated_at = NOW() WHERE id = $1;

-- name: RestoreFolder :exec
UPDATE folders SET trashed_at = NULL, updated_at = NOW() WHERE id = $1;

-- name: DeleteFolder :exec
DELETE FROM folders WHERE id = $1;

-- name: ListFoldersByParentID :many
SELECT * FROM folders
WHERE parent_id = $1 AND trashed_at IS NULL
ORDER BY name ASC;

-- name: ListFoldersByOwnerID :many
SELECT * FROM folders
WHERE owner_id = $1 AND trashed_at IS NULL
ORDER BY created_at DESC;

-- name: ListTrashedFoldersByOwnerID :many
SELECT * FROM folders
WHERE owner_id = $1 AND trashed_at IS NOT NULL
ORDER BY trashed_at DESC;

-- name: ListFoldersByPath :many
SELECT * FROM folders
WHERE path LIKE $1 || '%'
ORDER BY depth ASC;

-- name: CountChildFolders :one
SELECT COUNT(*) FROM folders
WHERE parent_id = $1 AND trashed_at IS NULL;

-- name: FolderExistsByName :one
SELECT EXISTS(
    SELECT 1 FROM folders
    WHERE parent_id = $1 AND owner_id = $2 AND name = $3 AND trashed_at IS NULL
);

-- name: GetFoldersToAutoDelete :many
SELECT * FROM folders
WHERE trashed_at IS NOT NULL AND trashed_at < $1;

-- name: UpdateFolderPath :exec
UPDATE folders SET
    path = $2,
    depth = $3,
    updated_at = NOW()
WHERE id = $1;

-- name: ListDescendantFolders :many
SELECT * FROM folders
WHERE path LIKE $1 || '/%'
ORDER BY depth ASC;
