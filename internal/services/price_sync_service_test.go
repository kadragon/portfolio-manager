package services_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/repositories"
	"github.com/kadragon/portfolio-manager/internal/services"
)

// trackingClient records which tickers were fetched.
type trackingClient struct {
	mu                sync.Mutex
	priceCalls        []string
	histCalls         []string
	quotesByTicker    map[string]services.PriceQuote
	histPriceByTicker map[string]float64
}

func (c *trackingClient) GetPrice(ticker, _ string) (services.PriceQuote, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.priceCalls = append(c.priceCalls, ticker)
	if q, ok := c.quotesByTicker[ticker]; ok {
		return q, nil
	}
	return services.PriceQuote{Symbol: ticker, Currency: "USD", Price: 100.0}, nil
}

func (c *trackingClient) GetHistoricalClose(ticker string, _ datex.Date, _ string) (float64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.histCalls = append(c.histCalls, ticker)
	if p, ok := c.histPriceByTicker[ticker]; ok {
		return p, nil
	}
	return 50.0, nil
}

func newSyncRepos(t *testing.T) (*repositories.StockPriceRepository, *repositories.StockRepository, *repositories.GroupRepository) {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return repositories.NewStockPriceRepository(q),
		repositories.NewStockRepository(q),
		repositories.NewGroupRepository(q)
}

func TestPriceSyncServiceSavesCurrentPrice(t *testing.T) {
	priceRepo, stockRepo, groupRepo := newSyncRepos(t)
	ctx := context.Background()

	g, err := groupRepo.Create(ctx, "test", 0)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	_, err = stockRepo.Create(ctx, "AAPL", g.ID)
	if err != nil {
		t.Fatalf("create stock: %v", err)
	}

	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"AAPL": {Symbol: "AAPL", Price: 200.0, Currency: "USD", Exchange: "NASD"},
		},
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo)
	svc.SyncOnce(ctx)

	sp, err := priceRepo.GetLatestByTicker(ctx, "AAPL")
	if err != nil {
		t.Fatalf("get latest: %v", err)
	}
	if sp == nil {
		t.Fatal("want saved price, got nil")
	}
	if !sp.Price.IsPositive() {
		t.Errorf("want positive price, got %v", sp.Price)
	}
}

func TestPriceSyncServiceSkipsHistoricalWhenPresent(t *testing.T) {
	priceRepo, stockRepo, groupRepo := newSyncRepos(t)
	ctx := context.Background()

	g, _ := groupRepo.Create(ctx, "test", 0)
	_, _ = stockRepo.Create(ctx, "VYM", g.ID)

	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"VYM": {Symbol: "VYM", Price: 160.0, Currency: "USD"},
		},
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo)

	// First sync: fills current price + all 4 historical periods.
	svc.SyncOnce(ctx)
	client.mu.Lock()
	firstHistCalls := len(client.histCalls)
	client.mu.Unlock()

	// Second sync: all historical dates now in DB → zero new hist calls.
	client.mu.Lock()
	client.histCalls = nil
	client.mu.Unlock()
	svc.SyncOnce(ctx)

	client.mu.Lock()
	secondHistCalls := len(client.histCalls)
	client.mu.Unlock()

	if firstHistCalls == 0 {
		t.Error("want hist calls on first sync, got 0")
	}
	if secondHistCalls != 0 {
		t.Errorf("want 0 hist calls on second sync (all cached), got %d", secondHistCalls)
	}
}

func TestPriceSyncServiceSkipsZeroPrice(t *testing.T) {
	priceRepo, stockRepo, groupRepo := newSyncRepos(t)
	ctx := context.Background()

	g, _ := groupRepo.Create(ctx, "test", 0)
	_, _ = stockRepo.Create(ctx, "ZERO", g.ID)

	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"ZERO": {Symbol: "ZERO", Price: 0, Currency: "USD"},
		},
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo)
	svc.SyncOnce(ctx)

	sp, _ := priceRepo.GetLatestByTicker(ctx, "ZERO")
	if sp != nil {
		t.Errorf("want no saved price for zero-price ticker, got %v", sp.Price)
	}
}

func TestPriceSyncServiceEmpty(t *testing.T) {
	priceRepo, stockRepo, _ := newSyncRepos(t)
	ctx := context.Background()

	client := &trackingClient{}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo)
	svc.SyncOnce(ctx)

	client.mu.Lock()
	calls := len(client.priceCalls)
	client.mu.Unlock()
	if calls != 0 {
		t.Errorf("want 0 calls with empty stock list, got %d", calls)
	}
}

func TestPriceSyncServiceStartStopsOnContextCancel(t *testing.T) {
	priceRepo, stockRepo, _ := newSyncRepos(t)

	client := &trackingClient{}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	go func() {
		svc.Start(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("Start did not stop after context cancel")
	}
}
