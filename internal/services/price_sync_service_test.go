package services_test

import (
	"context"
	"database/sql"
	"errors"
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
	histKeys          []string // "TICKER@YYYY-MM-DD", one per GetHistoricalClose call
	rangeCalls        []string
	quotesByTicker    map[string]services.PriceQuote
	histPriceByTicker map[string]float64
	// histEmptyDates marks ISO dates with no close (a market holiday), so a
	// GetHistoricalClose for one returns 0 the way KIS does.
	histEmptyDates map[string]bool
	// histErrDates marks ISO dates whose fetch fails outright — a transport or
	// rate-limit failure, which is not evidence the market was shut.
	histErrDates  map[string]bool
	rangeByTicker map[string][]services.HistoricalPricePoint
	// rangeKeys records "TICKER@START..END", one per GetHistoricalRange call.
	rangeKeys []string
	// synthesizeRange makes GetHistoricalRange answer from the same holiday and
	// price maps GetHistoricalClose uses, so a batched fetch can be tested against
	// the same closures. Off by default: the backfill tests drive the fake from
	// rangeByTicker instead.
	synthesizeRange bool
}

// rangeCallCount counts GetHistoricalRange calls made for one ticker.
func (c *trackingClient) rangeCallCount(ticker string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, t := range c.rangeCalls {
		if t == ticker {
			n++
		}
	}
	return n
}

// histKeyCount counts GetHistoricalClose calls made for SPY on one date.
func (c *trackingClient) histKeyCount(date datex.Date) int {
	return c.histKeyCountFor("SPY", date)
}

// histKeyCountFor counts GetHistoricalClose calls made for one ticker and date.
func (c *trackingClient) histKeyCountFor(ticker string, date datex.Date) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	want := ticker + "@" + date.ISO()
	n := 0
	for _, k := range c.histKeys {
		if k == want {
			n++
		}
	}
	return n
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

func (c *trackingClient) GetHistoricalClose(ticker string, date datex.Date, _ string) (float64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.histCalls = append(c.histCalls, ticker)
	c.histKeys = append(c.histKeys, ticker+"@"+date.ISO())
	if c.histErrDates[date.ISO()] {
		return 0, errors.New("simulated fetch failure")
	}
	if c.histEmptyDates[date.ISO()] {
		return 0, nil
	}
	if p, ok := c.histPriceByTicker[ticker]; ok {
		return p, nil
	}
	return 50.0, nil
}

func (c *trackingClient) GetHistoricalRange(ticker string, start, end datex.Date, _ string) ([]services.HistoricalPricePoint, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rangeCalls = append(c.rangeCalls, ticker)
	c.rangeKeys = append(c.rangeKeys, ticker+"@"+start.ISO()+".."+end.ISO())
	if !c.synthesizeRange {
		return c.rangeByTicker[ticker], nil
	}
	price := 50.0
	if p, ok := c.histPriceByTicker[ticker]; ok {
		price = p
	}
	var points []services.HistoricalPricePoint
	for d := start.Time; !d.After(end.Time); d = d.AddDate(0, 0, 1) {
		day := datex.FromTime(d)
		if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
			continue
		}
		if c.histEmptyDates[day.ISO()] {
			continue
		}
		points = append(points, services.HistoricalPricePoint{Date: day, Price: price})
	}
	return points, nil
}

func newSyncRepos(t *testing.T) (*repositories.StockPriceRepository, *repositories.StockRepository, *repositories.GroupRepository, *repositories.DepositRepository) {
	t.Helper()
	// Nothing here talks to KIS, so the rate-limit pacing is pure wall clock.
	services.SetSyncCallDelayForTest(t, 0)
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

// mustCreateDeposit fails the test on a rejected insert. deposit_date carries a
// unique index, so a discarded error silently leaves the sync with fewer deposits
// than the test believes it set up — and the assertions then pass vacuously.
func mustCreateDeposit(t *testing.T, ctx context.Context, repo *repositories.DepositRepository, date datex.Date) {
	t.Helper()
	if _, err := repo.Create(ctx, numeric.FromInt(100), date, sql.NullString{}); err != nil {
		t.Fatalf("create deposit %s: %v", date.ISO(), err)
	}
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
	mustCreateDeposit(t, ctx, depositRepo, firstDepositDate)

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
	mustCreateDeposit(t, ctx, depositRepo, firstDepositDate)

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

// Timing-matched benchmarks price EVERY deposit at its own date, so price-sync
// must checkpoint all of them — not just the first.
func TestPriceSyncServiceFetchesEveryDepositDateForBenchmarks(t *testing.T) {
	priceRepo, stockRepo, _, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	depositDates := []datex.Date{
		datex.New(2024, time.March, 12),
		datex.New(2025, time.July, 8),
		datex.New(2026, time.January, 15),
	}
	for _, d := range depositDates {
		mustCreateDeposit(t, ctx, depositRepo, d)
	}

	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"SPY":    {Symbol: "SPY", Price: 500.0, Currency: "USD", Exchange: "AMEX"},
			"QQQ":    {Symbol: "QQQ", Price: 450.0, Currency: "USD", Exchange: "NASD"},
			"226490": {Symbol: "226490", Price: 30000.0, Currency: "KRW"},
		},
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)

	// The timing-matched proxies are the ones that replay every deposit; the
	// dashboard-only benchmarks are covered by their own test below.
	for _, ticker := range []string{"226490", "360750", "368590"} {
		for _, d := range depositDates {
			sp, err := priceRepo.GetByTickerAndDate(ctx, ticker, d)
			if err != nil {
				t.Fatalf("get deposit-date price %s %s: %v", ticker, d.ISO(), err)
			}
			if sp == nil {
				t.Errorf("deposit-date price for %s on %s was not saved", ticker, d.ISO())
			}
		}
	}
}

// deposit_date carries a unique index, so two deposits never share a raw date.
// They collide only after prev-business-day adjustment: a Saturday and a Sunday
// deposit both resolve to the same Friday. That adjusted date is one checkpoint,
// so KIS must not be called twice for it.
func TestPriceSyncServiceFetchesAdjustedDuplicateDepositDateOnce(t *testing.T) {
	priceRepo, stockRepo, _, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	saturday := datex.New(2025, time.July, 5)
	sunday := datex.New(2025, time.July, 6)
	friday := datex.New(2025, time.July, 4) // what both adjust to

	for _, d := range []datex.Date{saturday, sunday} {
		mustCreateDeposit(t, ctx, depositRepo, d)
	}

	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"SPY": {Symbol: "SPY", Price: 500.0, Currency: "USD", Exchange: "AMEX"},
		},
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)

	if got := client.histKeyCount(friday); got != 1 {
		t.Errorf("GetHistoricalClose for SPY on %s called %d times, want 1", friday.ISO(), got)
	}
	for _, d := range []datex.Date{saturday, sunday} {
		if got := client.histKeyCount(d); got != 0 {
			t.Errorf("fetched the unadjusted weekend date %s %d times, want 0", d.ISO(), got)
		}
	}
}

// Deposit dates are a benchmark-only checkpoint set; held stocks sync the base
// 1y/6m/1m/1d checkpoints only.
func TestPriceSyncServiceSkipsDepositDatesForHeldStocks(t *testing.T) {
	priceRepo, stockRepo, groupRepo, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	g, _ := groupRepo.Create(ctx, "test", 0)
	_, _ = stockRepo.Create(ctx, "VYM", g.ID) // held stock, not a benchmark

	depositDates := []datex.Date{
		datex.New(2024, time.March, 12),
		datex.New(2025, time.July, 8),
	}
	for _, d := range depositDates {
		mustCreateDeposit(t, ctx, depositRepo, d)
	}

	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"VYM": {Symbol: "VYM", Price: 118.0, Currency: "USD", Exchange: "AMEX"},
			"SPY": {Symbol: "SPY", Price: 500.0, Currency: "USD", Exchange: "AMEX"},
		},
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)

	for _, d := range depositDates {
		// Precondition: the benchmark did take this checkpoint (guards against a
		// wall-clock run where a deposit date collides with a base checkpoint).
		spy, err := priceRepo.GetByTickerAndDate(ctx, "SPY", d)
		if err != nil {
			t.Fatalf("get SPY deposit-date price %s: %v", d.ISO(), err)
		}
		if spy == nil {
			continue
		}
		vym, err := priceRepo.GetByTickerAndDate(ctx, "VYM", d)
		if err != nil {
			t.Fatalf("get VYM deposit-date price %s: %v", d.ISO(), err)
		}
		if vym != nil {
			t.Errorf("held stock VYM fetched deposit-date price on %s (%v), want none", d.ISO(), vym.Price)
		}
	}
}

// prevBizDay only skips weekends, so a deposit on a market holiday lands on a day
// with no close. The sync walks back to the nearest earlier day that has one; the
// close is stored under that day, never mislabelled as the deposit date.
func TestPriceSyncServiceWalksBackFromHolidayDepositDate(t *testing.T) {
	priceRepo, stockRepo, _, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	// 2025-01-01 (Wed) is a holiday; 2024-12-31 (Tue) is the nearest open day.
	holiday := datex.New(2025, time.January, 1)
	fallback := datex.New(2024, time.December, 31)
	mustCreateDeposit(t, ctx, depositRepo, holiday)

	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"SPY": {Symbol: "SPY", Price: 500.0, Currency: "USD", Exchange: "AMEX"},
		},
		histEmptyDates: map[string]bool{holiday.ISO(): true},
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)

	if sp, err := priceRepo.GetByTickerAndDate(ctx, "SPY", holiday); err != nil {
		t.Fatalf("get SPY holiday price: %v", err)
	} else if sp != nil {
		t.Errorf("holiday %s saved price %v, want none (no close exists that day)", holiday.ISO(), sp.Price)
	}

	sp, err := priceRepo.GetByTickerAndDate(ctx, "SPY", fallback)
	if err != nil {
		t.Fatalf("get SPY fallback price: %v", err)
	}
	if sp == nil {
		t.Fatalf("fallback date %s has no price; the walk-back did not run", fallback.ISO())
	}
}

// The walk-back is bounded by a window, not an attempt count: a date with no
// close anywhere in it leaves no row rather than reaching back arbitrarily far.
func TestPriceSyncServiceBoundsHolidayWalkBack(t *testing.T) {
	priceRepo, stockRepo, _, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	depositDate := datex.New(2025, time.July, 8)
	mustCreateDeposit(t, ctx, depositRepo, depositDate)

	// Every day in and well beyond the walk-back window is empty.
	empty := map[string]bool{}
	for i := 0; i < 90; i++ {
		empty[datex.FromTime(depositDate.Time.AddDate(0, 0, -i)).ISO()] = true
	}
	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"SPY": {Symbol: "SPY", Price: 500.0, Currency: "USD", Exchange: "AMEX"},
		},
		histEmptyDates: empty,
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)

	for i := 0; i < 90; i++ {
		d := datex.FromTime(depositDate.Time.AddDate(0, 0, -i))
		if sp, err := priceRepo.GetByTickerAndDate(ctx, "SPY", d); err == nil && sp != nil {
			t.Fatalf("saved a price for %s despite no close being available", d.ISO())
		}
	}
	// The window is 14 days back from the deposit date; nothing older is touched.
	for i := 15; i < 90; i++ {
		d := datex.FromTime(depositDate.Time.AddDate(0, 0, -i))
		if got := client.histKeyCount(d); got != 0 {
			t.Errorf("walked back to %s (%d days), past the lookback window", d.ISO(), i)
		}
	}
}

// Korean closures cluster: KRX was shut 2025-10-03 (개천절) through 2025-10-09
// (한글날), five sessions. A deposit at the end of that run must still resolve, or
// every KRW benchmark proxy reports nil.
func TestPriceSyncServiceWalksBackOverHolidayCluster(t *testing.T) {
	priceRepo, stockRepo, _, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	depositDate := datex.New(2025, time.October, 9) // Thu, 한글날
	lastOpen := datex.New(2025, time.October, 2)    // Thu, the session before the run
	mustCreateDeposit(t, ctx, depositRepo, depositDate)

	empty := map[string]bool{}
	for _, d := range []datex.Date{
		datex.New(2025, time.October, 9),
		datex.New(2025, time.October, 8),
		datex.New(2025, time.October, 7),
		datex.New(2025, time.October, 6),
		datex.New(2025, time.October, 3),
	} {
		empty[d.ISO()] = true
	}
	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"360750": {Symbol: "360750", Price: 20000.0, Currency: "KRW"},
		},
		histEmptyDates: empty,
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)

	sp, err := priceRepo.GetByTickerAndDate(ctx, "360750", lastOpen)
	if err != nil {
		t.Fatalf("get 360750 price on %s: %v", lastOpen.ISO(), err)
	}
	if sp == nil {
		t.Fatalf("deposit on %s did not resolve back to %s; the cluster exhausted the walk-back",
			depositDate.ISO(), lastOpen.ISO())
	}
}

// A holiday deposit date never gets a row of its own, so without a coverage check
// every later sync would re-request it. Historical closes are fetch-once.
func TestPriceSyncServiceDoesNotRefetchResolvedHolidayDepositDate(t *testing.T) {
	priceRepo, stockRepo, _, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	holiday := datex.New(2025, time.January, 1)
	mustCreateDeposit(t, ctx, depositRepo, holiday)

	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"SPY": {Symbol: "SPY", Price: 500.0, Currency: "USD", Exchange: "AMEX"},
		},
		histEmptyDates: map[string]bool{holiday.ISO(): true},
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)
	first := client.histKeyCount(holiday)
	if first == 0 {
		t.Fatalf("first sync never tried the holiday date; test is inconclusive")
	}

	svc.SyncOnce(ctx)
	if got := client.histKeyCount(holiday); got != first {
		t.Errorf("second sync re-fetched the holiday date (%d → %d calls), want none", first, got)
	}
}

// A failed call is not evidence the market was shut. Walking back on it would
// multiply calls against an API already failing.
func TestPriceSyncServiceDoesNotWalkBackOnFetchError(t *testing.T) {
	priceRepo, stockRepo, _, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	depositDate := datex.New(2025, time.July, 8)
	mustCreateDeposit(t, ctx, depositRepo, depositDate)

	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"SPY": {Symbol: "SPY", Price: 500.0, Currency: "USD", Exchange: "AMEX"},
		},
		histErrDates: map[string]bool{depositDate.ISO(): true},
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)

	for i := 1; i <= 14; i++ {
		d := datex.FromTime(depositDate.Time.AddDate(0, 0, -i))
		if got := client.histKeyCount(d); got != 0 {
			t.Errorf("walked back to %s after a fetch error, want no walk-back", d.ISO())
		}
	}
}

// Only the timing-matched proxies replay every deposit. The dashboard-only
// benchmarks are measured from the first deposit alone, so fetching the rest for
// them would be pure cost against a rate-limited API.
func TestPriceSyncServiceFetchesOnlyFirstDepositForDashboardBenchmarks(t *testing.T) {
	priceRepo, stockRepo, _, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	first := datex.New(2024, time.March, 12)
	later := datex.New(2025, time.July, 8)
	for _, d := range []datex.Date{first, later} {
		mustCreateDeposit(t, ctx, depositRepo, d)
	}

	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"SPY":    {Symbol: "SPY", Price: 500.0, Currency: "USD", Exchange: "AMEX"},
			"360750": {Symbol: "360750", Price: 20000.0, Currency: "KRW"},
		},
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)

	// SPY is dashboard-only: first deposit yes, later deposit no.
	if sp, _ := priceRepo.GetByTickerAndDate(ctx, "SPY", first); sp == nil {
		t.Errorf("SPY missing the first-deposit checkpoint %s", first.ISO())
	}
	if sp, _ := priceRepo.GetByTickerAndDate(ctx, "SPY", later); sp != nil {
		t.Errorf("SPY fetched the later deposit date %s; dashboard-only benchmarks need only the first", later.ISO())
	}
	// 360750 is a timing-matched proxy: both dates.
	for _, d := range []datex.Date{first, later} {
		if sp, _ := priceRepo.GetByTickerAndDate(ctx, "360750", d); sp == nil {
			t.Errorf("timing-matched proxy 360750 missing checkpoint %s", d.ISO())
		}
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
	called := make(map[string]bool, len(client.priceCalls))
	for _, ticker := range client.priceCalls {
		called[ticker] = true
	}
	calls := len(client.priceCalls)
	client.mu.Unlock()
	// Both benchmark sets must be synced, not just the lump-sum one: a
	// timing-matched proxy with no cached price reports a nil return on an
	// otherwise healthy DB, and its current price would never refresh.
	for _, ticker := range []string{"SPY", "QQQ", "226490", "360750", "368590"} {
		if !called[ticker] {
			t.Errorf("benchmark %s not synced with empty stock list (calls: %v)", ticker, client.priceCalls)
		}
	}
	if calls != 5 {
		t.Errorf("want 5 benchmark calls with empty stock list, got %d", calls)
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

// A deposit date left unpriced blanks the whole timing-matched comparison, so
// SyncOnce reports the count rather than letting price-sync claim plain success.
func TestPriceSyncServiceReportsUnpricedBenchmarkDates(t *testing.T) {
	priceRepo, stockRepo, _, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	depositDate := datex.New(2025, time.July, 8)
	mustCreateDeposit(t, ctx, depositRepo, depositDate)

	// Nothing in the walk-back window has a close.
	empty := map[string]bool{}
	for i := 0; i < 40; i++ {
		empty[datex.FromTime(depositDate.Time.AddDate(0, 0, -i)).ISO()] = true
	}
	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"SPY": {Symbol: "SPY", Price: 500.0, Currency: "USD", Exchange: "AMEX"},
		},
		histEmptyDates: empty,
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)

	if got := svc.SyncOnce(ctx).UnpricedBenchmarkDates; got == 0 {
		t.Error("UnpricedBenchmarkDates = 0 with no close available, want a non-zero count")
	}
}

// A clean run reports nothing outstanding.
func TestPriceSyncServiceReportsNoUnpricedDatesOnCleanRun(t *testing.T) {
	priceRepo, stockRepo, _, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	mustCreateDeposit(t, ctx, depositRepo, datex.New(2025, time.July, 8))

	client := &trackingClient{
		quotesByTicker: map[string]services.PriceQuote{
			"SPY": {Symbol: "SPY", Price: 500.0, Currency: "USD", Exchange: "AMEX"},
		},
	}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)

	if got := svc.SyncOnce(ctx).UnpricedBenchmarkDates; got != 0 {
		t.Errorf("UnpricedBenchmarkDates = %d on a clean run, want 0", got)
	}
}

// timingMatchedTickers are the benchmarks that replay every deposit date, so they
// are the ones the batched prefetch is meant to spare from per-date calls.
var timingMatchedTickers = []string{"226490", "360750", "368590"}

// clusteredDepositDates are six monthly deposits that fall into exactly two
// 90-day windows: 2024-01-15 opens one that reaches 2024-04-14, 2024-05-15 opens
// the next.
func clusteredDepositDates() []datex.Date {
	return []datex.Date{
		datex.New(2024, time.January, 15),
		datex.New(2024, time.February, 15),
		datex.New(2024, time.March, 15),
		datex.New(2024, time.May, 15),
		datex.New(2024, time.June, 17),
		datex.New(2024, time.July, 15),
	}
}

func newClusteredDepositClient() *trackingClient {
	return &trackingClient{
		synthesizeRange: true,
		quotesByTicker: map[string]services.PriceQuote{
			"SPY":    {Symbol: "SPY", Price: 500.0, Currency: "USD", Exchange: "AMEX"},
			"QQQ":    {Symbol: "QQQ", Price: 450.0, Currency: "USD", Exchange: "NASD"},
			"226490": {Symbol: "226490", Price: 30000.0, Currency: "KRW"},
			"360750": {Symbol: "360750", Price: 20000.0, Currency: "KRW"},
			"368590": {Symbol: "368590", Price: 15000.0, Currency: "KRW"},
		},
	}
}

// Deposit dates clustered inside a 90-day window are what makes a long history
// slow: one GetHistoricalClose each, every one paying syncCallDelay. A single
// GetHistoricalRange covers the whole cluster.
func TestPriceSyncServiceBatchesClusteredDepositDates(t *testing.T) {
	priceRepo, stockRepo, _, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	depositDates := clusteredDepositDates()
	for _, d := range depositDates {
		mustCreateDeposit(t, ctx, depositRepo, d)
	}

	client := newClusteredDepositClient()
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)

	for _, ticker := range timingMatchedTickers {
		if got := client.rangeCallCount(ticker); got != 2 {
			t.Errorf("GetHistoricalRange for %s called %d times, want 2 (one per window)", ticker, got)
		}
		for _, d := range depositDates {
			if got := client.histKeyCountFor(ticker, d); got != 0 {
				t.Errorf("GetHistoricalClose for %s on %s called %d times, want 0 (batched)", ticker, d.ISO(), got)
			}
			sp, err := priceRepo.GetByTickerAndDate(ctx, ticker, d)
			if err != nil {
				t.Fatalf("get deposit-date price %s %s: %v", ticker, d.ISO(), err)
			}
			if sp == nil {
				t.Errorf("deposit-date price for %s on %s was not saved", ticker, d.ISO())
			}
		}
	}
}

// The batched span reaches back benchmarkDepositLookback before the first missing
// date, so a deposit that landed in a market closure has the preceding sessions in
// the same response and needs no walk-back calls of its own.
func TestPriceSyncServiceBatchCoversHolidayDepositDate(t *testing.T) {
	priceRepo, stockRepo, _, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	depositDates := clusteredDepositDates()
	for _, d := range depositDates {
		mustCreateDeposit(t, ctx, depositRepo, d)
	}
	holiday := datex.New(2024, time.February, 15)

	client := newClusteredDepositClient()
	client.histEmptyDates = map[string]bool{holiday.ISO(): true}
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)

	for _, ticker := range timingMatchedTickers {
		if got := client.histKeyCountFor(ticker, holiday); got != 0 {
			t.Errorf("GetHistoricalClose for %s on holiday %s called %d times, want 0", ticker, holiday.ISO(), got)
		}
		sp, err := priceRepo.GetOnOrBeforeDate(ctx, ticker, holiday)
		if err != nil {
			t.Fatalf("get on-or-before %s %s: %v", ticker, holiday.ISO(), err)
		}
		if sp == nil || !sp.Price.IsPositive() {
			t.Errorf("holiday deposit date %s for %s left unpriced", holiday.ISO(), ticker)
		}
	}
}

// A second pass over unchanged data must not re-request the ranges: every deposit
// date is already covered, so no window has two missing dates left to batch.
func TestPriceSyncServiceDoesNotRefetchBatchedDepositRanges(t *testing.T) {
	priceRepo, stockRepo, _, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	for _, d := range clusteredDepositDates() {
		mustCreateDeposit(t, ctx, depositRepo, d)
	}

	client := newClusteredDepositClient()
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)
	before := len(client.rangeCalls)
	svc.SyncOnce(ctx)

	if got := len(client.rangeCalls) - before; got != 0 {
		t.Errorf("second sync issued %d GetHistoricalRange calls, want 0", got)
	}
}

// Scattered deposits are the case the windowing rule exists to reject: a range
// call covering dates months apart would fetch a year of daily closes to reach a
// handful of them, so each stays on the single-date path.
func TestPriceSyncServiceDoesNotBatchScatteredDepositDates(t *testing.T) {
	priceRepo, stockRepo, groupRepo, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	g, err := groupRepo.Create(ctx, "test", 0)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := stockRepo.Create(ctx, "VYM", g.ID); err != nil {
		t.Fatalf("create stock: %v", err)
	}

	depositDates := []datex.Date{
		datex.New(2024, time.March, 12),
		datex.New(2025, time.July, 8),
		datex.New(2026, time.January, 15),
	}
	for _, d := range depositDates {
		mustCreateDeposit(t, ctx, depositRepo, d)
	}

	client := newClusteredDepositClient()
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)

	for _, ticker := range append(timingMatchedTickers, "VYM") {
		if got := client.rangeCallCount(ticker); got != 0 {
			t.Errorf("GetHistoricalRange for %s called %d times, want 0", ticker, got)
		}
	}
	for _, ticker := range timingMatchedTickers {
		for _, d := range depositDates {
			if got := client.histKeyCountFor(ticker, d); got != 1 {
				t.Errorf("GetHistoricalClose for %s on %s called %d times, want 1", ticker, d.ISO(), got)
			}
		}
	}
}

// Deposit dates are a benchmark-only checkpoint set, so the batching must not
// start pulling ranges for held stocks either.
func TestPriceSyncServiceDoesNotBatchDepositRangesForHeldStocks(t *testing.T) {
	priceRepo, stockRepo, groupRepo, depositRepo := newSyncRepos(t)
	ctx := context.Background()

	g, err := groupRepo.Create(ctx, "test", 0)
	if err != nil {
		t.Fatalf("create group: %v", err)
	}
	if _, err := stockRepo.Create(ctx, "VYM", g.ID); err != nil {
		t.Fatalf("create stock: %v", err)
	}
	for _, d := range clusteredDepositDates() {
		mustCreateDeposit(t, ctx, depositRepo, d)
	}

	client := newClusteredDepositClient()
	svc := services.NewPriceSyncService(client, priceRepo, stockRepo, depositRepo)
	svc.SyncOnce(ctx)

	if got := client.rangeCallCount("VYM"); got != 0 {
		t.Errorf("GetHistoricalRange for held stock VYM called %d times, want 0", got)
	}
	// SPY and QQQ are dashboard-only proxies: one deposit date each, never a cluster.
	for _, ticker := range []string{"SPY", "QQQ"} {
		if got := client.rangeCallCount(ticker); got != 0 {
			t.Errorf("GetHistoricalRange for dashboard-only %s called %d times, want 0", ticker, got)
		}
	}
}
