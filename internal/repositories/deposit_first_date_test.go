package repositories_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/repositories"
)

func TestDepositGetFirstDateEmpty(t *testing.T) {
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	r := repositories.NewDepositRepository(q)

	got, err := r.GetFirstDepositDate(context.Background())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil on empty table, got %v", got)
	}
}

func TestDepositGetFirstDateReturnsEarliest(t *testing.T) {
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	r := repositories.NewDepositRepository(q)
	ctx := context.Background()

	amt, _ := numeric.FromString("100000")
	d1, _ := datex.ParseDate("2026-01-01")
	d2, _ := datex.ParseDate("2025-06-15") // earlier
	d3, _ := datex.ParseDate("2026-03-01")

	for _, d := range []datex.Date{d1, d2, d3} {
		if _, err := r.Create(ctx, amt, d, sql.NullString{}); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	got, err := r.GetFirstDepositDate(ctx)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got == nil {
		t.Fatal("expected date, got nil")
	}
	if got.ISO() != d2.ISO() {
		t.Errorf("first date = %s, want %s", got.ISO(), d2.ISO())
	}
}
