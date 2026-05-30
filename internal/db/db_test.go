package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// TestScanProductionFormats inserts rows using the exact literal forms found in
// the production database (hex32 UUID, integer-stored decimal, datetime with
// offset and microseconds) and verifies the custom types scan them.
func TestScanProductionFormats(t *testing.T) {
	sqlDB, _, err := OpenMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()
	ctx := context.Background()

	_, err = sqlDB.ExecContext(ctx,
		`INSERT INTO accounts (id, name, cash_balance, kis_account_no, kis_api_key_id, created_at, updated_at)
		 VALUES (?,?,?,?,?,?,?)`,
		"5aa9c13b1ac74c0dabe2a9ee715b0f84", "주식계좌", 968616, nil, nil,
		"2026-01-03 13:21:44.873677+00:00", "2026-01-03 13:21:44.873677+00:00",
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	var (
		id   uuidx.UUID
		name string
		cash numeric.Decimal
		kis  sql.NullString
		ca   ktime.Time
	)
	row := sqlDB.QueryRowContext(ctx,
		`SELECT id, name, cash_balance, kis_account_no, created_at FROM accounts WHERE id = ?`,
		"5aa9c13b1ac74c0dabe2a9ee715b0f84")
	if err := row.Scan(&id, &name, &cash, &kis, &ca); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if id.Hex() != "5aa9c13b1ac74c0dabe2a9ee715b0f84" {
		t.Errorf("id = %q", id.Hex())
	}
	if cash.String() != "968616" {
		t.Errorf("cash = %q, want 968616", cash.String())
	}
	if kis.Valid {
		t.Errorf("kis_account_no should be NULL, got %q", kis.String)
	}
	if ca.Year() != 2026 || ca.Nanosecond() != 873677000 {
		t.Errorf("created_at = %v", ca.Time)
	}
}

// TestDecimalValuerRoundTrip verifies fractional decimals survive a write via
// the Valuer and a read via the Scanner.
func TestDecimalValuerRoundTrip(t *testing.T) {
	sqlDB, _, err := OpenMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()
	ctx := context.Background()

	frac, err := numeric.FromString("12.677005")
	if err != nil {
		t.Fatalf("from string: %v", err)
	}
	now := ktime.Now()
	id := uuidx.New()
	_, err = sqlDB.ExecContext(ctx,
		`INSERT INTO accounts (id, name, cash_balance, created_at, updated_at) VALUES (?,?,?,?,?)`,
		id, "frac", frac, now, now,
	)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}

	var back numeric.Decimal
	if err := sqlDB.QueryRowContext(ctx, `SELECT cash_balance FROM accounts WHERE id = ?`, id).Scan(&back); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if back.String() != "12.677005" {
		t.Fatalf("round trip = %q, want 12.677005", back.String())
	}
}

// TestForeignKeysEnforced confirms the FK pragma is active.
func TestForeignKeysEnforced(t *testing.T) {
	sqlDB, _, err := OpenMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sqlDB.Close()
	now := ktime.Now()
	// stocks.group_id references a non-existent group -> must fail.
	_, err = sqlDB.ExecContext(context.Background(),
		`INSERT INTO stocks (id, ticker, name, group_id, created_at, updated_at) VALUES (?,?,?,?,?,?)`,
		uuidx.New(), "005930", "", uuidx.New(), now, now,
	)
	if err == nil {
		t.Fatal("expected foreign key violation, got nil")
	}
}
