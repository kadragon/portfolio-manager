package repositories_test

import (
	"context"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/repositories"
)

func newStockRepos(t *testing.T) (*repositories.GroupRepository, *repositories.StockRepository) {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { sqlDB.Close() })
	return repositories.NewGroupRepository(q), repositories.NewStockRepository(q)
}

func TestStockAssetClassRoundTrip(t *testing.T) {
	groups, stocks := newStockRepos(t)
	ctx := context.Background()

	g, _ := groups.Create(ctx, "국내배당", 15.0)
	s, err := stocks.Create(ctx, "0052D0", g.ID)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if s.AssetClass != nil {
		t.Fatalf("fresh asset_class should be nil, got %v", *s.AssetClass)
	}

	updated, err := stocks.UpdateAssetClass(ctx, s.ID, "etf")
	if err != nil {
		t.Fatalf("update asset class: %v", err)
	}
	if updated.AssetClass == nil || *updated.AssetClass != "etf" {
		t.Fatalf("asset_class = %v, want etf", updated.AssetClass)
	}

	got, err := stocks.GetByID(ctx, s.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.AssetClass == nil || *got.AssetClass != "etf" {
		t.Fatalf("reloaded asset_class = %v, want etf", got.AssetClass)
	}
}

// TestStockAssetClassClear proves an empty value resets a previously-set
// asset class back to NULL (the "미분류" option in the edit UI).
func TestStockAssetClassClear(t *testing.T) {
	groups, stocks := newStockRepos(t)
	ctx := context.Background()

	g, _ := groups.Create(ctx, "국내배당", 15.0)
	s, _ := stocks.Create(ctx, "0052D0", g.ID)
	if _, err := stocks.UpdateAssetClass(ctx, s.ID, "etf"); err != nil {
		t.Fatalf("set asset class: %v", err)
	}

	cleared, err := stocks.UpdateAssetClass(ctx, s.ID, "")
	if err != nil {
		t.Fatalf("clear asset class: %v", err)
	}
	if cleared.AssetClass != nil {
		t.Fatalf("asset_class after clear = %v, want nil", *cleared.AssetClass)
	}

	got, err := stocks.GetByID(ctx, s.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.AssetClass != nil {
		t.Fatalf("reloaded asset_class after clear = %v, want nil", *got.AssetClass)
	}
}

func TestStockCreateAndList(t *testing.T) {
	groups, stocks := newStockRepos(t)
	ctx := context.Background()

	g, _ := groups.Create(ctx, "성장주", 30.0)
	s, err := stocks.Create(ctx, "AAPL", g.ID)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if s.Ticker != "AAPL" || s.GroupID != g.ID {
		t.Fatalf("created = %+v", s)
	}
	if s.Exchange != nil {
		t.Fatalf("exchange should be nil, got %v", s.Exchange)
	}
	if s.Name != "" {
		t.Fatalf("name should be empty, got %q", s.Name)
	}

	all, err := stocks.ListByGroup(ctx, g.ID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 1 || all[0].ID != s.ID {
		t.Fatalf("list = %+v", all)
	}
}

func TestStockListAllAndEmpty(t *testing.T) {
	groups, stocks := newStockRepos(t)
	ctx := context.Background()

	all, err := stocks.ListAll(ctx)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected empty, got %d", len(all))
	}

	g, _ := groups.Create(ctx, "g", 0)
	if _, err := stocks.Create(ctx, "MSFT", g.ID); err != nil {
		t.Fatalf("create MSFT: %v", err)
	}
	if _, err := stocks.Create(ctx, "GOOGL", g.ID); err != nil {
		t.Fatalf("create GOOGL: %v", err)
	}
	all, _ = stocks.ListAll(ctx)
	if len(all) != 2 {
		t.Fatalf("expected 2, got %d", len(all))
	}
}

func TestStockGetByIDAbsentReturnsNil(t *testing.T) {
	groups, stocks := newStockRepos(t)
	ctx := context.Background()

	g, _ := groups.Create(ctx, "g", 0)
	s, _ := stocks.Create(ctx, "X", g.ID)

	got, err := stocks.GetByID(ctx, s.ID)
	if err != nil || got == nil || got.Ticker != "X" {
		t.Fatalf("get = %v, %v", got, err)
	}

	if err := stocks.Delete(ctx, s.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	absent, err := stocks.GetByID(ctx, s.ID)
	if err != nil {
		t.Fatalf("get absent: %v", err)
	}
	if absent != nil {
		t.Fatalf("expected nil for deleted stock, got %+v", absent)
	}
}

func TestStockGetByTicker(t *testing.T) {
	groups, stocks := newStockRepos(t)
	ctx := context.Background()

	g, _ := groups.Create(ctx, "g", 0)
	if _, err := stocks.Create(ctx, "NVDA", g.ID); err != nil {
		t.Fatalf("create: %v", err)
	}

	s, err := stocks.GetByTicker(ctx, "NVDA")
	if err != nil || s == nil || s.Ticker != "NVDA" {
		t.Fatalf("by ticker = %v, %v", s, err)
	}
	missing, err := stocks.GetByTicker(ctx, "MISSING")
	if err != nil || missing != nil {
		t.Fatalf("expected nil for missing ticker, got %v, %v", missing, err)
	}
}

func TestStockUpdateTicker(t *testing.T) {
	groups, stocks := newStockRepos(t)
	ctx := context.Background()

	g, _ := groups.Create(ctx, "g", 0)
	s, _ := stocks.Create(ctx, "OLD", g.ID)

	updated, err := stocks.UpdateTicker(ctx, s.ID, "NEW")
	if err != nil || updated.Ticker != "NEW" {
		t.Fatalf("update ticker: %v %+v", err, updated)
	}
}

func TestStockUpdateGroup(t *testing.T) {
	groups, stocks := newStockRepos(t)
	ctx := context.Background()

	g1, _ := groups.Create(ctx, "g1", 0)
	g2, _ := groups.Create(ctx, "g2", 0)
	s, _ := stocks.Create(ctx, "MOVE", g1.ID)

	updated, err := stocks.UpdateGroup(ctx, s.ID, g2.ID)
	if err != nil || updated.GroupID != g2.ID {
		t.Fatalf("update group: %v %+v", err, updated)
	}
}

func TestStockUpdateExchangeAndName(t *testing.T) {
	groups, stocks := newStockRepos(t)
	ctx := context.Background()

	g, _ := groups.Create(ctx, "g", 0)
	s, _ := stocks.Create(ctx, "T", g.ID)

	withExchange, err := stocks.UpdateExchange(ctx, s.ID, "NASDAQ")
	if err != nil || withExchange.Exchange == nil || *withExchange.Exchange != "NASDAQ" {
		t.Fatalf("update exchange: %v %+v", err, withExchange)
	}

	withName, err := stocks.UpdateName(ctx, s.ID, "테슬라")
	if err != nil || withName.Name != "테슬라" {
		t.Fatalf("update name: %v %+v", err, withName)
	}
}

func TestStockListInsertionOrder(t *testing.T) {
	groups, stocks := newStockRepos(t)
	ctx := context.Background()

	g, _ := groups.Create(ctx, "g", 0)
	tickers := []string{"A", "B", "C"}
	for _, tk := range tickers {
		if _, err := stocks.Create(ctx, tk, g.ID); err != nil {
			t.Fatalf("create %s: %v", tk, err)
		}
	}

	all, _ := stocks.ListByGroup(ctx, g.ID)
	if len(all) != 3 {
		t.Fatalf("len = %d", len(all))
	}
	for i, tk := range tickers {
		if all[i].Ticker != tk {
			t.Errorf("position %d = %q, want %q (insertion order)", i, all[i].Ticker, tk)
		}
	}
}
