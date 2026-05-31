package repositories_test

import (
	"context"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/db"
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

	rec, err := r.Create(ctx, "005930", "buy", 10, "KRW", "filled", "ok", "KRX",
		map[string]any{"rt_cd": "0"})
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
}

func TestOrderExecutionListRecent(t *testing.T) {
	r := newOrderExecRepo(t)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, err := r.Create(ctx, "AAPL", "sell", i+1, "USD", "filled", "", "NASD", nil)
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
