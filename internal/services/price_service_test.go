package services_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"testing"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/repositories"
	"github.com/kadragon/portfolio-manager/internal/services"
)

// --- mock PriceClient ---

type mockPriceClient struct {
	mu        sync.Mutex
	quotes    map[string]services.PriceQuote
	getCalls  int
	histCalls int
}

func (m *mockPriceClient) GetPrice(ticker, _ string) (services.PriceQuote, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getCalls++
	if q, ok := m.quotes[ticker]; ok {
		return q, nil
	}
	return services.PriceQuote{}, fmt.Errorf("ticker %s not found", ticker)
}

func (m *mockPriceClient) GetHistoricalClose(_ string, _ datex.Date, _ string) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.histCalls++
	return 70000, nil
}

// --- helpers ---

func newPriceRepo(t *testing.T) *repositories.StockPriceRepository {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return repositories.NewStockPriceRepository(q)
}

func TestGetStockPriceWithMockClient(t *testing.T) {
	r := newPriceRepo(t)
	client := &mockPriceClient{
		quotes: map[string]services.PriceQuote{
			"005930": {
				Symbol:   "005930",
				Name:     "삼성전자",
				Price:    74000,
				Currency: "KRW",
				Exchange: "",
			},
		},
	}
	svc := services.NewPriceService(r, client)
	ctx := context.Background()

	price, currency, name, _ := svc.GetStockPrice(ctx, "005930", "")
	if !price.IsPositive() {
		t.Errorf("want positive price, got %v", price)
	}
	if currency != "KRW" {
		t.Errorf("want KRW, got %s", currency)
	}
	if name != "삼성전자" {
		t.Errorf("want '삼성전자', got %s", name)
	}
	if client.getCalls != 1 {
		t.Errorf("want 1 GetPrice call, got %d", client.getCalls)
	}

	// Second call: must return from in-memory cache (no extra client hit).
	price2, _, _, _ := svc.GetStockPrice(ctx, "005930", "")
	if !price2.Equal(price.Decimal) {
		t.Errorf("second call: price changed — want %v, got %v", price, price2)
	}
	if client.getCalls != 1 {
		t.Errorf("want 1 total GetPrice call (cache hit), got %d", client.getCalls)
	}
}

func TestGetStockPriceConcurrentAccess(t *testing.T) {
	r := newPriceRepo(t)
	client := &mockPriceClient{
		quotes: map[string]services.PriceQuote{
			"005930": {
				Symbol:   "005930",
				Name:     "삼성전자",
				Price:    74000,
				Currency: "KRW",
				Exchange: "",
			},
		},
	}
	svc := services.NewPriceService(r, client)
	ctx := context.Background()
	start := make(chan struct{})

	var wg sync.WaitGroup
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			price, currency, _, _ := svc.GetStockPrice(ctx, "005930", "")
			if !price.IsPositive() || currency != "KRW" {
				t.Errorf("unexpected price result: %s %s", price.String(), currency)
			}
		}()
	}
	close(start)
	wg.Wait()
}

func TestGetStockPriceNilClient(t *testing.T) {
	r := newPriceRepo(t)
	svc := services.NewPriceService(r, nil)
	ctx := context.Background()

	price, _, _, _ := svc.GetStockPrice(ctx, "005930", "")
	if !price.IsZero() {
		t.Errorf("want zero price with nil client and empty cache, got %v", price)
	}
}

func TestGetCachedPrice(t *testing.T) {
	r := newPriceRepo(t)
	ctx := context.Background()

	d, _ := datex.ParseDate("2026-01-15")
	p, _ := numeric.FromString("74000")
	_, err := r.Save(ctx, "005930", d, p, "KRW", "삼성전자", sql.NullString{})
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	svc := services.NewPriceService(r, nil)
	got := svc.GetCachedPrice(ctx, "005930", d)
	if got == nil {
		t.Fatal("want non-nil cached price, got nil")
	}
	if !got.Price.IsPositive() {
		t.Errorf("want positive price, got %v", got.Price)
	}
	if got.Ticker != "005930" {
		t.Errorf("want ticker 005930, got %s", got.Ticker)
	}
}

func TestGetStockChangeRatesEmpty(t *testing.T) {
	r := newPriceRepo(t)
	svc := services.NewPriceService(r, nil)
	ctx := context.Background()

	// nil client → GetStockChangeRates returns nil immediately.
	result := svc.GetStockChangeRates(ctx, "005930", "", []string{"1d"})
	if result != nil {
		t.Errorf("want nil result with nil client, got %v", result)
	}
}

func TestGetStockChangeRatesEmptyPeriods(t *testing.T) {
	r := newPriceRepo(t)
	client := &mockPriceClient{
		quotes: map[string]services.PriceQuote{
			"005930": {Symbol: "005930", Price: 74000, Currency: "KRW"},
		},
	}
	svc := services.NewPriceService(r, client)
	ctx := context.Background()

	// Empty periods → returns nil (no valid normalized periods).
	result := svc.GetStockChangeRates(ctx, "005930", "", []string{})
	if result != nil {
		t.Errorf("want nil for empty periods, got %v", result)
	}
}

func TestGetStockChangeRatesSmoke(t *testing.T) {
	r := newPriceRepo(t)
	client := &mockPriceClient{
		quotes: map[string]services.PriceQuote{
			"005930": {Symbol: "005930", Name: "삼성전자", Price: 74000, Currency: "KRW"},
		},
	}
	svc := services.NewPriceService(r, client)
	ctx := context.Background()

	result := svc.GetStockChangeRates(ctx, "005930", "", []string{"1d", "1m"})
	if result == nil {
		t.Fatal("want non-nil result, got nil")
	}
	if _, ok := result["1d"]; !ok {
		t.Error("want '1d' key in result")
	}
	if _, ok := result["1m"]; !ok {
		t.Error("want '1m' key in result")
	}
}
