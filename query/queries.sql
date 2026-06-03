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
INSERT INTO stocks (id, ticker, group_id, exchange, created_at, updated_at, name, asset_class)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, ticker, group_id, exchange, created_at, updated_at, name, asset_class;

-- name: ListStocksByGroup :many
SELECT id, ticker, group_id, exchange, created_at, updated_at, name, asset_class FROM stocks WHERE group_id = ?;

-- name: ListAllStocks :many
SELECT id, ticker, group_id, exchange, created_at, updated_at, name, asset_class FROM stocks;

-- name: GetStockByID :one
SELECT id, ticker, group_id, exchange, created_at, updated_at, name, asset_class FROM stocks WHERE id = ?;

-- name: GetStockByTicker :one
SELECT id, ticker, group_id, exchange, created_at, updated_at, name, asset_class FROM stocks WHERE ticker = ?;

-- name: UpdateStockTicker :one
UPDATE stocks SET ticker = ?, updated_at = ? WHERE id = ?
RETURNING id, ticker, group_id, exchange, created_at, updated_at, name, asset_class;

-- name: UpdateStockGroup :one
UPDATE stocks SET group_id = ?, updated_at = ? WHERE id = ?
RETURNING id, ticker, group_id, exchange, created_at, updated_at, name, asset_class;

-- name: UpdateStockExchange :one
UPDATE stocks SET exchange = ?, updated_at = ? WHERE id = ?
RETURNING id, ticker, group_id, exchange, created_at, updated_at, name, asset_class;

-- name: UpdateStockName :one
UPDATE stocks SET name = ?, updated_at = ? WHERE id = ?
RETURNING id, ticker, group_id, exchange, created_at, updated_at, name, asset_class;

-- name: UpdateStockAssetClass :one
UPDATE stocks SET asset_class = ?, updated_at = ? WHERE id = ?
RETURNING id, ticker, group_id, exchange, created_at, updated_at, name, asset_class;

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
SET name = ?, cash_balance = ?, kis_account_no = ?, kis_api_key_id = ?, account_type = ?, updated_at = ?
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

-- Deposit queries (Phase 5).

-- name: CreateDeposit :one
INSERT INTO deposits (id, amount, deposit_date, note, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListDeposits :many
SELECT * FROM deposits ORDER BY deposit_date DESC;

-- name: GetDepositByID :one
SELECT * FROM deposits WHERE id = ?;

-- name: GetDepositByDate :one
SELECT * FROM deposits WHERE deposit_date = ?;

-- name: UpdateDeposit :one
UPDATE deposits SET amount = ?, deposit_date = ?, note = ?, updated_at = ? WHERE id = ?
RETURNING *;

-- name: UpdateDepositWithoutNote :one
UPDATE deposits SET amount = ?, deposit_date = ?, updated_at = ? WHERE id = ?
RETURNING *;

-- name: DeleteDeposit :exec
DELETE FROM deposits WHERE id = ?;

-- Phase 6 queries.

-- name: ListAllHoldings :many
SELECT * FROM holdings;

-- name: GetFirstDepositDate :one
SELECT deposit_date FROM deposits ORDER BY deposit_date ASC LIMIT 1;

-- name: GetStockPriceByTickerAndDate :one
SELECT * FROM stock_prices WHERE ticker = ? AND price_date = ?;

-- name: GetLatestStockPriceByTicker :one
SELECT * FROM stock_prices WHERE ticker = ? ORDER BY price_date DESC LIMIT 1;

-- name: GetStockPriceOnOrBeforeDate :one
SELECT * FROM stock_prices WHERE ticker = ? AND price_date <= ? ORDER BY price_date DESC LIMIT 1;

-- name: UpsertStockPrice :one
INSERT INTO stock_prices (id, ticker, price, currency, name, exchange, price_date, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(ticker, price_date) DO UPDATE SET
    price=excluded.price,
    currency=excluded.currency,
    name=CASE WHEN excluded.name != '' THEN excluded.name ELSE stock_prices.name END,
    exchange=excluded.exchange,
    updated_at=excluded.updated_at
RETURNING *;

-- Phase 7 queries.

-- name: CreateOrderExecution :one
INSERT INTO order_executions (id, ticker, side, quantity, currency, exchange, status, message, raw_response, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListRecentOrderExecutions :many
SELECT * FROM order_executions ORDER BY created_at DESC LIMIT ?;
