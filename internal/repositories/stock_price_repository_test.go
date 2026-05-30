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

func newStockPriceRepo(t *testing.T) *repositories.StockPriceRepository {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return repositories.NewStockPriceRepository(q)
}

func TestStockPriceSaveAndGet(t *testing.T) {
	r := newStockPriceRepo(t)
	ctx := context.Background()

	d, _ := datex.ParseDate("2026-01-15")
	price, _ := numeric.FromString("85000")
	saved, err := r.Save(ctx, "005930", d, price, "KRW", "삼성전자", sql.NullString{})
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if saved.Ticker != "005930" {
		t.Errorf("ticker = %s", saved.Ticker)
	}

	got, err := r.GetByTickerAndDate(ctx, "005930", d)
	if err != nil || got == nil {
		t.Fatalf("get: got=%v err=%v", got, err)
	}
	if got.ID != saved.ID {
		t.Errorf("id mismatch")
	}
}

func TestStockPriceAbsentReturnsNil(t *testing.T) {
	r := newStockPriceRepo(t)
	ctx := context.Background()

	d, _ := datex.ParseDate("2026-03-01")
	got, err := r.GetByTickerAndDate(ctx, "AAPL", d)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %+v", got)
	}
}

func TestStockPriceUpsertPreservesName(t *testing.T) {
	r := newStockPriceRepo(t)
	ctx := context.Background()

	d, _ := datex.ParseDate("2026-02-01")
	p1, _ := numeric.FromString("100")
	_, err := r.Save(ctx, "AAPL", d, p1, "USD", "Apple Inc.", sql.NullString{})
	if err != nil {
		t.Fatalf("first save: %v", err)
	}

	// Upsert with empty name → should preserve existing name
	p2, _ := numeric.FromString("105")
	updated, err := r.Save(ctx, "AAPL", d, p2, "USD", "", sql.NullString{})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if updated.Name != "Apple Inc." {
		t.Errorf("name not preserved: %s", updated.Name)
	}
	if !updated.Price.Equal(p2.Decimal) {
		t.Errorf("price not updated: %v", updated.Price)
	}
}
