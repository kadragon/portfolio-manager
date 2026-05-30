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

-- Stock queries (Phase 2).

-- name: CreateStock :one
INSERT INTO stocks (id, ticker, group_id, exchange, created_at, updated_at, name)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id, ticker, group_id, exchange, created_at, updated_at, name;

-- name: ListStocksByGroup :many
SELECT id, ticker, group_id, exchange, created_at, updated_at, name FROM stocks WHERE group_id = ?;

-- name: ListAllStocks :many
SELECT id, ticker, group_id, exchange, created_at, updated_at, name FROM stocks;

-- name: GetStockByID :one
SELECT id, ticker, group_id, exchange, created_at, updated_at, name FROM stocks WHERE id = ?;

-- name: GetStockByTicker :one
SELECT id, ticker, group_id, exchange, created_at, updated_at, name FROM stocks WHERE ticker = ?;

-- name: UpdateStockTicker :one
UPDATE stocks SET ticker = ?, updated_at = ? WHERE id = ?
RETURNING id, ticker, group_id, exchange, created_at, updated_at, name;

-- name: UpdateStockGroup :one
UPDATE stocks SET group_id = ?, updated_at = ? WHERE id = ?
RETURNING id, ticker, group_id, exchange, created_at, updated_at, name;

-- name: UpdateStockExchange :one
UPDATE stocks SET exchange = ?, updated_at = ? WHERE id = ?
RETURNING id, ticker, group_id, exchange, created_at, updated_at, name;

-- name: UpdateStockName :one
UPDATE stocks SET name = ?, updated_at = ? WHERE id = ?
RETURNING id, ticker, group_id, exchange, created_at, updated_at, name;

-- name: DeleteStock :exec
DELETE FROM stocks WHERE id = ?;

-- Account queries (Phase 3).

-- name: CreateAccount :one
INSERT INTO accounts (id, name, cash_balance, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: ListAccounts :many
SELECT * FROM accounts;

-- name: GetAccountByID :one
SELECT * FROM accounts WHERE id = ?;

-- name: UpdateAccountNameCash :one
UPDATE accounts SET name = ?, cash_balance = ?, updated_at = ? WHERE id = ?
RETURNING *;

-- name: UpdateAccount :one
UPDATE accounts
SET name = ?, cash_balance = ?, kis_account_no = ?, kis_api_key_id = ?, updated_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteAccount :exec
DELETE FROM accounts WHERE id = ?;

-- name: DeleteHoldingsByAccount :exec
DELETE FROM holdings WHERE account_id = ?;

-- Holding queries (Phase 4).

-- name: CreateHolding :one
INSERT INTO holdings (id, account_id, stock_id, quantity, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListHoldingsByAccount :many
SELECT * FROM holdings WHERE account_id = ?;

-- name: GetHoldingByID :one
SELECT * FROM holdings WHERE id = ?;

-- name: UpdateHolding :one
UPDATE holdings SET quantity = ?, updated_at = ? WHERE id = ?
RETURNING *;

-- name: DeleteHolding :exec
DELETE FROM holdings WHERE id = ?;
