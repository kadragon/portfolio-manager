package db

import (
	"context"
	"database/sql"
	"testing"

	dbsqlc "github.com/kadragon/portfolio-manager/internal/db/sqlc"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// columnNames returns the column names of a table via PRAGMA table_info.
func columnNames(t *testing.T, sqlDB *sql.DB, table string) []string {
	t.Helper()
	rows, err := sqlDB.QueryContext(context.Background(), "PRAGMA table_info("+table+")")
	if err != nil {
		t.Fatalf("pragma table_info(%s): %v", table, err)
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var (
			cid     int
			name    string
			ctype   string
			notnull int
			dflt    sql.NullString
			pk      int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			t.Fatalf("scan table_info: %v", err)
		}
		cols = append(cols, name)
	}
	return cols
}

func countCol(cols []string, target string) int {
	n := 0
	for _, c := range cols {
		if c == target {
			n++
		}
	}
	return n
}

// TestMigrateIdempotent builds an OLD-schema database (no account_type /
// asset_class), inserts a row, then runs migrate twice. Each new column must
// exist exactly once and the pre-existing row must be preserved.
func TestMigrateIdempotent(t *testing.T) {
	sqlDB, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)&cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	ctx := context.Background()
	accountID := uuidx.New()

	// Old schema: accounts without account_type, stocks without asset_class.
	const oldSchema = `
CREATE TABLE "accounts" (
    "id" TEXT NOT NULL PRIMARY KEY,
    "name" TEXT NOT NULL,
    "cash_balance" DECIMAL(10, 10) NOT NULL,
    "created_at" DATETIME NOT NULL,
    "updated_at" DATETIME NOT NULL,
    "kis_account_no" TEXT,
    "kis_api_key_id" INTEGER
);
CREATE TABLE "stocks" (
    "id" TEXT NOT NULL PRIMARY KEY,
    "ticker" TEXT NOT NULL,
    "group_id" TEXT NOT NULL,
    "exchange" TEXT,
    "created_at" DATETIME NOT NULL,
    "updated_at" DATETIME NOT NULL,
    "name" TEXT NOT NULL
);`
	if _, err := sqlDB.ExecContext(ctx, oldSchema); err != nil {
		t.Fatalf("old schema: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx,
		`INSERT INTO accounts (id, name, cash_balance, created_at, updated_at) VALUES (?,?,?,?,?)`,
		accountID, "기존계좌", 1000, "2026-01-01 00:00:00+00:00", "2026-01-01 00:00:00+00:00",
	); err != nil {
		t.Fatalf("seed account: %v", err)
	}

	for i := 0; i < 2; i++ {
		if err := migrate(ctx, sqlDB); err != nil {
			t.Fatalf("migrate pass %d: %v", i, err)
		}
	}

	accCols := columnNames(t, sqlDB, "accounts")
	if n := countCol(accCols, "account_type"); n != 1 {
		t.Errorf("accounts.account_type count = %d, want 1 (cols: %v)", n, accCols)
	}
	for _, column := range []string{"cash_balance_krw", "cash_balance_usd"} {
		if n := countCol(accCols, column); n != 1 {
			t.Errorf("accounts.%s count = %d, want 1 (cols: %v)", column, n, accCols)
		}
	}
	stkCols := columnNames(t, sqlDB, "stocks")
	if n := countCol(stkCols, "asset_class"); n != 1 {
		t.Errorf("stocks.asset_class count = %d, want 1 (cols: %v)", n, stkCols)
	}

	var name string
	if err := sqlDB.QueryRowContext(ctx, `SELECT name FROM accounts WHERE id = ?`, accountID).Scan(&name); err != nil {
		t.Fatalf("read seeded row: %v", err)
	}
	if name != "기존계좌" {
		t.Errorf("seeded row name = %q, want 기존계좌", name)
	}
	var krwCash, usdCash any
	if err := sqlDB.QueryRowContext(ctx,
		`SELECT cash_balance_krw, cash_balance_usd FROM accounts WHERE id = ?`, accountID,
	).Scan(&krwCash, &usdCash); err != nil {
		t.Fatalf("read migrated cash balances: %v", err)
	}
	if krwCash != nil || usdCash != nil {
		t.Fatalf("migrated legacy cash balances = %v, %v; want NULL, NULL", krwCash, usdCash)
	}
	legacy, err := dbsqlc.New(sqlDB).GetAccountByID(ctx, accountID)
	if err != nil {
		t.Fatalf("read migrated account through sqlc: %v", err)
	}
	if legacy.CashBalanceKrw != nil || legacy.CashBalanceUsd != nil {
		t.Fatalf("sqlc migrated cash balances = %v, %v; want nil, nil",
			legacy.CashBalanceKrw, legacy.CashBalanceUsd)
	}
}

// TestOpenMemoryHasNewColumns proves a fresh DB (schema.sql + migrate) exposes
// the new columns exactly once — no duplicate from ALTER on top of CREATE.
func TestOpenMemoryHasNewColumns(t *testing.T) {
	sqlDB, _, err := OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	defer sqlDB.Close()
	if n := countCol(columnNames(t, sqlDB, "accounts"), "account_type"); n != 1 {
		t.Errorf("fresh accounts.account_type count = %d, want 1", n)
	}
	for _, column := range []string{"cash_balance_krw", "cash_balance_usd"} {
		if n := countCol(columnNames(t, sqlDB, "accounts"), column); n != 1 {
			t.Errorf("fresh accounts.%s count = %d, want 1", column, n)
		}
	}
	if n := countCol(columnNames(t, sqlDB, "stocks"), "asset_class"); n != 1 {
		t.Errorf("fresh stocks.asset_class count = %d, want 1", n)
	}
	for _, column := range []string{"order_type", "price"} {
		if n := countCol(columnNames(t, sqlDB, "order_executions"), column); n != 1 {
			t.Errorf("fresh order_executions.%s count = %d, want 1", column, n)
		}
	}
}

// TestMigrateAddsOrderExecutionColumns exercises the production upgrade path: a
// DB whose order_executions table predates order_type/price gains both columns
// on migrate, and an existing row survives with NULL values.
func TestMigrateAddsOrderExecutionColumns(t *testing.T) {
	sqlDB, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)&cache=shared")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()
	sqlDB.SetMaxOpenConns(1)
	ctx := context.Background()
	rowID := uuidx.New()

	const oldSchema = `
CREATE TABLE "order_executions" (
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
);`
	if _, err := sqlDB.ExecContext(ctx, oldSchema); err != nil {
		t.Fatalf("old schema: %v", err)
	}
	if _, err := sqlDB.ExecContext(ctx,
		`INSERT INTO order_executions (id, ticker, side, quantity, currency, status, message, created_at) VALUES (?,?,?,?,?,?,?,?)`,
		rowID, "005930", "buy", 10, "KRW", "filled", "ok", "2026-01-01 00:00:00+00:00",
	); err != nil {
		t.Fatalf("seed order execution: %v", err)
	}

	for i := 0; i < 2; i++ {
		if err := migrate(ctx, sqlDB); err != nil {
			t.Fatalf("migrate pass %d: %v", i, err)
		}
	}

	cols := columnNames(t, sqlDB, "order_executions")
	for _, column := range []string{"order_type", "price"} {
		if n := countCol(cols, column); n != 1 {
			t.Errorf("order_executions.%s count = %d, want 1 (cols: %v)", column, n, cols)
		}
	}

	var orderType, price any
	if err := sqlDB.QueryRowContext(ctx,
		`SELECT order_type, price FROM order_executions WHERE id = ?`, rowID,
	).Scan(&orderType, &price); err != nil {
		t.Fatalf("read migrated row: %v", err)
	}
	if orderType != nil || price != nil {
		t.Fatalf("migrated legacy order_type/price = %v, %v; want NULL, NULL", orderType, price)
	}
}
