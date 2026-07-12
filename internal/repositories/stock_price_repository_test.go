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

func TestStockPriceDelete(t *testing.T) {
	r := newStockPriceRepo(t)
	ctx := context.Background()
	d, _ := datex.ParseDate("2026-01-15")
	price, _ := numeric.FromString("85000")
	if _, err := r.Save(ctx, "005930", d, price, "KRW", "삼성전자", sql.NullString{}); err != nil {
		t.Fatalf("save: %v", err)
	}
	deleted, err := r.Delete(ctx, "005930", d)
	if err != nil || !deleted {
		t.Fatalf("delete existing: deleted=%v err=%v", deleted, err)
	}
	deleted, err = r.Delete(ctx, "005930", d)
	if err != nil || deleted {
		t.Fatalf("delete missing: deleted=%v err=%v", deleted, err)
	}
}

func TestStockPriceGetLatestByTicker(t *testing.T) {
	r := newStockPriceRepo(t)
	ctx := context.Background()

	d, _ := datex.ParseDate("2026-01-03")
	price, _ := numeric.FromString("74000")
	_, err := r.Save(ctx, "005930", d, price, "KRW", "삼성전자", sql.NullString{})
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	sp, err := r.GetLatestByTicker(ctx, "005930")
	if err != nil {
		t.Fatalf("GetLatestByTicker: %v", err)
	}
	if sp == nil {
		t.Fatal("expected non-nil")
	}
	if sp.Ticker != "005930" {
		t.Errorf("Ticker = %q, want 005930", sp.Ticker)
	}
}

func TestStockPriceGetLatestByTickerNotFound(t *testing.T) {
	r := newStockPriceRepo(t)
	sp, err := r.GetLatestByTicker(context.Background(), "NOTEXIST")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sp != nil {
		t.Errorf("expected nil for missing ticker, got %+v", sp)
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

func TestStockPriceGetOnOrBeforeDate(t *testing.T) {
	r := newStockPriceRepo(t)
	ctx := context.Background()

	d1, _ := datex.ParseDate("2026-01-10")
	d2, _ := datex.ParseDate("2026-01-20")
	p1, _ := numeric.FromString("100")
	p2, _ := numeric.FromString("110")
	if _, err := r.Save(ctx, "005930", d1, p1, "KRW", "삼성전자", sql.NullString{}); err != nil {
		t.Fatalf("save d1: %v", err)
	}
	if _, err := r.Save(ctx, "005930", d2, p2, "KRW", "삼성전자", sql.NullString{}); err != nil {
		t.Fatalf("save d2: %v", err)
	}

	// A date between the two rows must resolve to the earlier one.
	mid, _ := datex.ParseDate("2026-01-15")
	got, err := r.GetOnOrBeforeDate(ctx, "005930", mid)
	if err != nil || got == nil {
		t.Fatalf("get on-or-before: got=%v err=%v", got, err)
	}
	if got.PriceDate.Format("2006-01-02") != "2026-01-10" {
		t.Errorf("price date = %v, want 2026-01-10", got.PriceDate)
	}

	// A date before all rows must return nil, not an error.
	early, _ := datex.ParseDate("2026-01-01")
	got, err = r.GetOnOrBeforeDate(ctx, "005930", early)
	if err != nil {
		t.Fatalf("early: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil before first price, got %+v", got)
	}
}

func TestStockPriceListByTickerRange(t *testing.T) {
	r := newStockPriceRepo(t)
	ctx := context.Background()

	dates := []string{"2026-01-10", "2026-01-15", "2026-01-20", "2026-01-25"}
	for _, ds := range dates {
		d, _ := datex.ParseDate(ds)
		p, _ := numeric.FromString("100")
		if _, err := r.Save(ctx, "005930", d, p, "KRW", "삼성전자", sql.NullString{}); err != nil {
			t.Fatalf("save %s: %v", ds, err)
		}
	}

	from, _ := datex.ParseDate("2026-01-12")
	to, _ := datex.ParseDate("2026-01-22")
	got, err := r.ListByTickerRange(ctx, "005930", from, to, 10)
	if err != nil {
		t.Fatalf("list by range: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 rows within range, got %d", len(got))
	}
	if got[0].PriceDate.Format("2006-01-02") != "2026-01-20" {
		t.Errorf("row 0 date = %v, want newest-first 2026-01-20", got[0].PriceDate)
	}
	if got[1].PriceDate.Format("2006-01-02") != "2026-01-15" {
		t.Errorf("row 1 date = %v, want 2026-01-15", got[1].PriceDate)
	}

	// limit caps the result even when more rows are in range.
	wide, _ := datex.ParseDate("2026-01-01")
	got, err = r.ListByTickerRange(ctx, "005930", wide, to, 1)
	if err != nil {
		t.Fatalf("list with limit: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected limit=1 to cap result, got %d rows", len(got))
	}
}
