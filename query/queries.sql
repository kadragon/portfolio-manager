-- Group queries (Phase 1).

-- name: CreateGroup :one
INSERT INTO groups (id, name, target_percentage, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListGroups :many
SELECT * FROM groups;

-- name: GetGroup :one
SELECT * FROM groups WHERE id = ?;

-- name: UpdateGroup :one
UPDATE groups
SET name = ?, target_percentage = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteGroup :exec
DELETE FROM groups WHERE id = ?;
