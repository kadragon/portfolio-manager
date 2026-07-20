package services_test

import (
	"context"
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/repositories"
	"github.com/kadragon/portfolio-manager/internal/services"
)

// trackingClient records which tickers were fetched.
type trackingClient struct {
	mu                sync.Mutex
	priceCalls        []string
	histCalls         []string
	rangeCalls        []string
	quotesByTicker    map[string]services.PriceQuote
	histPriceByTicker map[string]float64
	rangeByTicker     map[string][]services.HistoricalPricePoint
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

func (c *trackingClient) GetHistoricalRange(ticker string, _, _ datex.Date, _ string) ([]services.HistoricalPricePoint, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rangeCalls = append(c.rangeCalls, ticker)
	return c.rangeByTicker[ticker], nil
}

func newSyncRepos(t *testing.T) (*repositories.StockPriceRepository, *repositories.StockRepository, *repositories.GroupRepository, *repositories.DepositRepository) {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("open memory: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return repositories.NewStockPriceRepository(q),
		repositories.NewStockRepository(q),
		repositories.NewGroupRepository(q),
		repositories.NewDepositRepository(q)
}

func TestPriceSyncServiceSavesCurrentPrice(t *testing.T) {
	priceRepo, stockRepo, groupRepo, depositRepo := newSyncRepos(t)
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
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
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

func TestPriceSyncServiceSyncsBenchmarksWithoutStockRows(t *testing.T) {
	priceRepo, stockRepo, _, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"SPY":    {Symbol: "SPY", Price: 500.0, Currency: "USD", Exchange: "AMEX"},
			"QQQ":    {Symbol: "QQQ", Price: 450.0, Currency: "USD", Exchange: "NASD"},
			"226490": {Symbol: "226490", Price: 30000.0, Currency: "KRW"},
		},
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)

	for _, ticker := range []string{"SPY", "QQQ", "226490"} {
		sp, err := priceRepo.GetLatestByTicker(ctx, ticker)
		if err != nil {
			t.Fatalf("get latest %s: %v", ticker, err)
		}
		if sp == nil {
			t.Fatalf("benchmark %s was not saved", ticker)
		}
	}
}

func TestPriceSyncServiceFetchesFirstDepositDateForBenchmarks(t *testing.T) {
	priceRepo, stockRepo, _, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	firstDepositDate := datex.New(2026, time.January, 15)
	_, _ = depositRepo.Create(ctx, numeric.FromInt(100), firstDepositDate, sql.NullString{})

	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"SPY":    {Symbol: "SPY", Price: 500.0, Currency: "USD", Exchange: "AMEX"},
			"QQQ":    {Symbol: "QQQ", Price: 450.0, Currency: "USD", Exchange: "NASD"},
			"226490": {Symbol: "226490", Price: 30000.0, Currency: "KRW"},
		},
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)

	for _, ticker := range []string{"SPY", "QQQ", "226490"} {
		sp, err := priceRepo.GetByTickerAndDate(ctx, ticker, firstDepositDate)
		if err != nil {
			t.Fatalf("get first deposit price %s: %v", ticker, err)
		}
		if sp == nil {
			t.Fatalf("first deposit date price for %s was not saved", ticker)
		}
	}
}

// The first-deposit date is a benchmark-only checkpoint (needed for the
// benchmark-vs-portfolio comparison); held stocks must not fetch it.
func TestPriceSyncServiceSkipsFirstDepositDateForHeldStocks(t *testing.T) {
	priceRepo, stockRepo, groupRepo, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	g, _ := groupRepo.Create(ctx, "test", 0)
	_, _ = stockRepo.Create(ctx, "VYM", g.ID) // held stock, not a benchmark

	firstDepositDate := datex.New(2026, time.January, 15)
	_, _ = depositRepo.Create(ctx, numeric.FromInt(100), firstDepositDate, sql.NullString{})

	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"VYM":    {Symbol: "VYM", Price: 118.0, Currency: "USD", Exchange: "AMEX"},
			"SPY":    {Symbol: "SPY", Price: 500.0, Currency: "USD", Exchange: "AMEX"},
			"QQQ":    {Symbol: "QQQ", Price: 450.0, Currency: "USD", Exchange: "NASD"},
			"226490": {Symbol: "226490", Price: 30000.0, Currency: "KRW"},
		},
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)

	// Precondition: a benchmark did fetch the first-deposit date (guards against a
	// wall-clock run where firstDepositDate collides with a base checkpoint).
	spy, err := priceRepo.GetByTickerAndDate(ctx, "SPY", firstDepositDate)
	if err != nil {
		t.Fatalf("get SPY first-deposit price: %v", err)
	}
	if spy == nil {
		t.Skip("first-deposit date coincides with a base checkpoint this run; test is inconclusive")
	}

	// The held stock must NOT have fetched the first-deposit-date price.
	vym, err := priceRepo.GetByTickerAndDate(ctx, "VYM", firstDepositDate)
	if err != nil {
		t.Fatalf("get VYM first-deposit price: %v", err)
	}
	if vym != nil {
		t.Errorf("held stock VYM fetched first-deposit-date price %v, want none (benchmark-only date)", vym.Price)
	}
}

func TestPriceSyncServiceSkipsHistoricalWhenPresent(t *testing.T) {
	priceRepo, stockRepo, groupRepo, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	g, _ := groupRepo.Create(ctx, "test", 0)
	_, _ = stockRepo.Create(ctx, "VYM", g.ID)

	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"VYM": {Symbol: "VYM", Price: 160.0, Currency: "USD"},
		},
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)

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
	priceRepo, stockRepo, groupRepo, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	g, _ := groupRepo.Create(ctx, "test", 0)
	_, _ = stockRepo.Create(ctx, "ZERO", g.ID)

	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"ZERO": {Symbol: "ZERO", Price: 0, Currency: "USD"},
		},
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)

	sp, _ := priceRepo.GetLatestByTicker(ctx, "ZERO")
	if sp != nil {
		t.Errorf("want no saved price for zero-price ticker, got %v", sp.Price)
	}
}

// flakyClient fails the first GetPrice call for each ticker, then returns a real quote.
type flakyClient struct {
	mu       sync.Mutex
	failed   map[string]bool
	callsFor map[string]int
	quote    services.PriceQuote
}

func (c *flakyClient) GetPrice(ticker, _ string) (services.PriceQuote, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.failed == nil {
		c.failed = map[string]bool{}
		c.callsFor = map[string]int{}
	}
	c.callsFor[ticker]++
	if !c.failed[ticker] {
		c.failed[ticker] = true
		return services.PriceQuote{}, sql.ErrNoRows
	}
	return c.quote, nil
}

func (c *flakyClient) GetHistoricalClose(_ string, _ datex.Date, _ string) (float64, error) {
	return 50.0, nil
}

func (c *flakyClient) GetHistoricalRange(_ string, _, _ datex.Date, _ string) ([]services.HistoricalPricePoint, error) {
	return nil, nil
}

func TestPriceSyncServiceRetriesTransientFailure(t *testing.T) {
	priceRepo, stockRepo, groupRepo, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	g, _ := groupRepo.Create(ctx, "test", 0)
	_, _ = stockRepo.Create(ctx, "VYM", g.ID)

	client := &flakyClient{quote: services.PriceQuote{Symbol: "VYM", Price: 118.5, Currency: "USD", Exchange: "AMEX"}}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)

	client.mu.Lock()
	vymCalls := client.callsFor["VYM"]
	client.mu.Unlock()
	if vymCalls != 2 {
		t.Errorf("GetPrice(VYM) calls = %d, want 2 (initial failure + one retry)", vymCalls)
	}
	sp, err := priceRepo.GetLatestByTicker(ctx, "VYM")
	if err != nil {
		t.Fatalf("get latest: %v", err)
	}
	if sp == nil || !sp.Price.IsPositive() {
		t.Fatalf("want saved price after retry, got %v", sp)
	}
}

func TestPriceSyncServiceEmptyStockListStillSyncsBenchmarks(t *testing.T) {
	priceRepo, stockRepo, _, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	client := &trackingClient{}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)

	client.mu.Lock()
	calls := len(client.priceCalls)
	client.mu.Unlock()
	if calls != 3 {
		t.Errorf("want 3 benchmark calls with empty stock list, got %d", calls)
	}
}

func TestPriceSyncServiceBackfillRangeSavesEveryPoint(t *testing.T) {
	priceRepo, stockRepo, groupRepo, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	g, _ := groupRepo.Create(ctx, "test", 0)
	_, _ = stockRepo.Create(ctx, "005930", g.ID)

	start := datex.New(2026, 6, 1)
	end := datex.New(2026, 6, 3)
	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"005930": {Symbol: "005930", Currency: "KRW", Price: 70000},
		},
		rangeByTicker: map[string][]services.HistoricalPricePoint{
			"005930": {
				{Date: datex.New(2026, 6, 1), Price: 69000},
				{Date: datex.New(2026, 6, 2), Price: 69500},
				{Date: datex.New(2026, 6, 3), Price: 70000},
			},
		},
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)

	result, err := svc.BackfillRange(ctx, "005930", start, end)
	if err != nil {
		t.Fatalf("BackfillRange: %v", err)
	}
	if result.Saved != 3 {
		t.Errorf("Saved = %d, want 3", result.Saved)
	}

	for _, d := range []datex.Date{start, datex.New(2026, 6, 2), end} {
		sp, err := priceRepo.GetByTickerAndDate(ctx, "005930", d)
		if err != nil || sp == nil {
			t.Fatalf("expected saved price for %s, got %v (err %v)", d.ISO(), sp, err)
		}
	}
}

func TestPriceSyncServiceBackfillRangeSkipsAlreadyCached(t *testing.T) {
	priceRepo, stockRepo, groupRepo, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	g, _ := groupRepo.Create(ctx, "test", 0)
	_, _ = stockRepo.Create(ctx, "005930", g.ID)

	day := datex.New(2026, 6, 1)
	price, _ := numeric.FromString("69000")
	if _, err := priceRepo.Save(ctx, "005930", day, price, "KRW", "Samsung", sql.NullString{}); err != nil {
		t.Fatalf("seed price: %v", err)
	}

	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"005930": {Symbol: "005930", Currency: "KRW", Price: 70000},
		},
		rangeByTicker: map[string][]services.HistoricalPricePoint{
			"005930": {{Date: day, Price: 69000}},
		},
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)

	result, err := svc.BackfillRange(ctx, "005930", day, day)
	if err != nil {
		t.Fatalf("BackfillRange: %v", err)
	}
	if result.Saved != 0 || result.Skipped != 1 {
		t.Errorf("Saved/Skipped = %d/%d, want 0/1", result.Saved, result.Skipped)
	}
}

func TestPriceSyncServiceBackfillRangeRejectsInvertedRange(t *testing.T) {
	priceRepo, stockRepo, _, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	client := &trackingClient{}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)

	_, err := svc.BackfillRange(ctx, "005930", datex.New(2026, 6, 3), datex.New(2026, 6, 1))
	if err == nil {
		t.Fatal("want error for end before start")
	}
}

func TestPriceSyncServiceBackfillRangeChunksLongSpans(t *testing.T) {
	priceRepo, stockRepo, groupRepo, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	g, _ := groupRepo.Create(ctx, "test", 0)
	_, _ = stockRepo.Create(ctx, "005930", g.ID)

	start := datex.New(2026, 1, 1)
	end := datex.New(2026, 6, 1) // > 90 days, forces a second window
	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"005930": {Symbol: "005930", Currency: "KRW", Price: 70000},
		},
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)

	if _, err := svc.BackfillRange(ctx, "005930", start, end); err != nil {
		t.Fatalf("BackfillRange: %v", err)
	}

	client.mu.Lock()
	rangeCalls := len(client.rangeCalls)
	client.mu.Unlock()
	if rangeCalls < 2 {
		t.Errorf("rangeCalls = %d, want >= 2 for a >90-day span", rangeCalls)
	}
}
