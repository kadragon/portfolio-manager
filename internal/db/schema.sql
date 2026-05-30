-- SQLite schema mirroring the Peewee models in services/database.py.
--
-- Parity rules (so the existing production .data/portfolio.db keeps working and
-- a fresh database is structurally identical):
--   * Column order matches the production database exactly. Columns added by
--     Peewee migrations (stocks.name, accounts.kis_account_no/kis_api_key_id)
--     come LAST, matching ALTER TABLE ordering. sqlc scans `SELECT *` in schema
--     order, so this order must equal the on-disk order.
--   * Index names match Peewee's generated names, so CREATE INDEX IF NOT EXISTS
--     is a no-op on the production database (no duplicate indexes).
--   * No SQL DEFAULT clauses: Peewee applies field defaults in Python, never in
--     DDL. The application always supplies every column on insert.
--   * UUID PKs are 32-char hex TEXT; DECIMAL columns have NUMERIC affinity;
--     DATE/DATETIME are TEXT. Timestamps are written by the app (KST).

CREATE TABLE IF NOT EXISTS "groups" (
    "id" TEXT NOT NULL PRIMARY KEY,
    "name" TEXT NOT NULL,
    "target_percentage" REAL NOT NULL,
    "created_at" DATETIME NOT NULL,
    "updated_at" DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS "stocks" (
    "id" TEXT NOT NULL PRIMARY KEY,
    "ticker" TEXT NOT NULL,
    "group_id" TEXT NOT NULL,
    "exchange" TEXT,
    "created_at" DATETIME NOT NULL,
    "updated_at" DATETIME NOT NULL,
    "name" TEXT NOT NULL,
    FOREIGN KEY ("group_id") REFERENCES "groups" ("id") ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS "stockmodel_group_id" ON "stocks" ("group_id");
CREATE INDEX IF NOT EXISTS "stockmodel_ticker" ON "stocks" ("ticker");

CREATE TABLE IF NOT EXISTS "accounts" (
    "id" TEXT NOT NULL PRIMARY KEY,
    "name" TEXT NOT NULL,
    "cash_balance" DECIMAL(10, 10) NOT NULL,
    "created_at" DATETIME NOT NULL,
    "updated_at" DATETIME NOT NULL,
    "kis_account_no" TEXT,
    "kis_api_key_id" INTEGER
);

CREATE TABLE IF NOT EXISTS "holdings" (
    "id" TEXT NOT NULL PRIMARY KEY,
    "account_id" TEXT NOT NULL,
    "stock_id" TEXT NOT NULL,
    "quantity" DECIMAL(10, 10) NOT NULL,
    "created_at" DATETIME NOT NULL,
    "updated_at" DATETIME NOT NULL,
    FOREIGN KEY ("account_id") REFERENCES "accounts" ("id") ON DELETE CASCADE,
    FOREIGN KEY ("stock_id") REFERENCES "stocks" ("id") ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS "holdingmodel_account_id" ON "holdings" ("account_id");
CREATE INDEX IF NOT EXISTS "holdingmodel_stock_id" ON "holdings" ("stock_id");

CREATE TABLE IF NOT EXISTS "deposits" (
    "id" TEXT NOT NULL PRIMARY KEY,
    "amount" DECIMAL(10, 10) NOT NULL,
    "deposit_date" DATE NOT NULL,
    "note" TEXT,
    "created_at" DATETIME NOT NULL,
    "updated_at" DATETIME NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS "depositmodel_deposit_date" ON "deposits" ("deposit_date");

CREATE TABLE IF NOT EXISTS "stock_prices" (
    "id" TEXT NOT NULL PRIMARY KEY,
    "ticker" TEXT NOT NULL,
    "price" DECIMAL(10, 10) NOT NULL,
    "currency" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "exchange" TEXT,
    "price_date" DATE NOT NULL,
    "created_at" DATETIME NOT NULL,
    "updated_at" DATETIME NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS "stockpricemodel_ticker_price_date" ON "stock_prices" ("ticker", "price_date");
CREATE INDEX IF NOT EXISTS "stockpricemodel_ticker" ON "stock_prices" ("ticker");

CREATE TABLE IF NOT EXISTS "order_executions" (
    "id" TEXT NOT NULL PRIMARY KEY,
    "ticker" TEXT NOT NULL,
    "side" TEXT NOT NULL,
    "quantity" INTEGER NOT NULL,
    "currency" TEXT NOT NULL,
    "exchange" TEXT,
    "status" TEXT NOT NULL,
    "message" TEXT NOT NULL,
    "raw_response" TEXT,
    "created_at" DATETIME NOT NULL
);
CREATE INDEX IF NOT EXISTS "orderexecutionmodel_created_at" ON "order_executions" ("created_at");

CREATE TABLE IF NOT EXISTS "investor_flows" (
    "id" TEXT NOT NULL PRIMARY KEY,
    "ticker" TEXT NOT NULL,
    "flow_date" DATE NOT NULL,
    "foreign_net_qty" INTEGER NOT NULL,
    "institution_net_qty" INTEGER NOT NULL,
    "individual_net_qty" INTEGER NOT NULL,
    "foreign_net_krw" INTEGER NOT NULL,
    "institution_net_krw" INTEGER NOT NULL,
    "individual_net_krw" INTEGER NOT NULL,
    "created_at" DATETIME NOT NULL,
    "updated_at" DATETIME NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS "investorflowmodel_ticker_flow_date" ON "investor_flows" ("ticker", "flow_date");
CREATE INDEX IF NOT EXISTS "investorflowmodel_ticker" ON "investor_flows" ("ticker");
