package db

import (
	"context"
	"database/sql"
	"testing"
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
		"acc1", "기존계좌", 1000, "2026-01-01 00:00:00+00:00", "2026-01-01 00:00:00+00:00",
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
	stkCols := columnNames(t, sqlDB, "stocks")
	if n := countCol(stkCols, "asset_class"); n != 1 {
		t.Errorf("stocks.asset_class count = %d, want 1 (cols: %v)", n, stkCols)
	}

	var name string
	if err := sqlDB.QueryRowContext(ctx, `SELECT name FROM accounts WHERE id = ?`, "acc1").Scan(&name); err != nil {
		t.Fatalf("read seeded row: %v", err)
	}
	if name != "기존계좌" {
		t.Errorf("seeded row name = %q, want 기존계좌", name)
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
	if n := countCol(columnNames(t, sqlDB, "stocks"), "asset_class"); n != 1 {
		t.Errorf("fresh stocks.asset_class count = %d, want 1", n)
	}
}
