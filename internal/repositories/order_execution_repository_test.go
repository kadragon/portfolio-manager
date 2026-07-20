package repositories_test

import (
	"context"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/repositories"
)

func newOrderExecRepo(t *testing.T) *repositories.OrderExecutionRepository {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return repositories.NewOrderExecutionRepository(q)
}

func TestOrderExecutionCreate(t *testing.T) {
	r := newOrderExecRepo(t)
	ctx := context.Background()

	limitPrice := numeric.FromInt(74000)
	rec, err := r.Create(ctx, "005930", "buy", 10, "KRW", "filled", "ok", "KRX",
		"limit", &limitPrice, map[string]any{"rt_cd": "0"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rec.Ticker != "005930" {
		t.Errorf("Ticker = %q, want 005930", rec.Ticker)
	}
	if rec.Side != "buy" {
		t.Errorf("Side = %q, want buy", rec.Side)
	}
	if rec.Quantity != 10 {
		t.Errorf("Quantity = %d, want 10", rec.Quantity)
	}
	// A limit order must round-trip its type and price out of storage.
	if rec.OrderType != "limit" {
		t.Errorf("OrderType = %q, want limit", rec.OrderType)
	}
	if rec.Price == nil || rec.Price.String() != "74000" {
		t.Errorf("Price = %v, want 74000", rec.Price)
	}
}

func TestOrderExecutionCreateMarketOrder(t *testing.T) {
	r := newOrderExecRepo(t)
	ctx := context.Background()

	rec, err := r.Create(ctx, "AAPL", "buy", 1, "USD", "filled", "ok", "NASD",
		"market", nil, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rec.OrderType != "market" {
		t.Errorf("OrderType = %q, want market", rec.OrderType)
	}
	if rec.Price != nil {
		t.Errorf("Price = %v, want nil for market order", rec.Price)
	}

	// The persisted row must read back the same way through the list path.
	got, err := r.ListRecent(ctx, 1)
	if err != nil {
		t.Fatalf("ListRecent: %v", err)
	}
	if len(got) != 1 || got[0].OrderType != "market" || got[0].Price != nil {
		t.Fatalf("round-trip market order = %+v", got)
	}
}

func TestOrderExecutionListRecent(t *testing.T) {
	r := newOrderExecRepo(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := r.Create(ctx, "AAPL", "sell", i+1, "USD", "filled", "", "NASD", "market", nil, nil)
		if err != nil {
			t.Fatalf("Create[%d]: %v", i, err)
		}
	}

	recs, err := r.ListRecent(ctx, 2)
	if err != nil {
		t.Fatalf("ListRecent: %v", err)
	}
	if len(recs) != 2 {
		t.Errorf("ListRecent(2) returned %d records, want 2", len(recs))
	}
}

func TestOrderExecutionListRecentEmpty(t *testing.T) {
	r := newOrderExecRepo(t)
	recs, err := r.ListRecent(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListRecent empty: %v", err)
	}
	if len(recs) != 0 {
		t.Errorf("expected empty, got %d", len(recs))
	}
}

func TestOrderExecutionListFilters(t *testing.T) {
	r := newOrderExecRepo(t)
	ctx := context.Background()
	if _, err := r.Create(ctx, "AAPL", "buy", 1, "USD", "filled", "ok", "NASD", "market", nil, nil); err != nil {
		t.Fatalf("Create filled: %v", err)
	}
	if _, err := r.Create(ctx, "005930", "sell", 2, "KRW", "failed", "rejected", "KRX", "market", nil, nil); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	records, err := r.List(ctx, "005930", "failed", 10)
	if err != nil {
		t.Fatalf("List filters: %v", err)
	}
	if len(records) != 1 || records[0].Ticker != "005930" || records[0].Status != "failed" {
		t.Fatalf("unexpected filtered records: %+v", records)
	}
}
