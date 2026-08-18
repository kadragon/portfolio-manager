package services_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/services"
)

func newPortfolioContainer(t *testing.T) *container.Container {
	t.Helper()
	sqlDB, q, err := db.OpenMemory()
	if err != nil {
		t.Fatalf("OpenMemory: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return container.NewWithQueries(sqlDB, q)
}

func TestHasPriceServiceFalse(t *testing.T) {
	c := newPortfolioContainer(t)
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, nil, nil)
	if ps.HasPriceService() {
		t.Error("HasPriceService() with nil priceService should be false")
	}
}

func TestHasPriceServiceTrue(t *testing.T) {
	c := newPortfolioContainer(t)
	priceService := services.NewPriceService(c.StockPrices)
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, nil)
	if !ps.HasPriceService() {
		t.Error("HasPriceService() with non-nil priceService should be true")
	}
}

func TestGetHoldingsByGroupEmpty(t *testing.T) {
	c := newPortfolioContainer(t)
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, nil, nil)
	groups, err := ps.GetHoldingsByGroup(context.Background())
	if err != nil {
		t.Fatalf("GetHoldingsByGroup: %v", err)
	}
	if len(groups) != 0 {
		t.Errorf("expected empty, got %d groups", len(groups))
	}
}

func TestGetHoldingsByGroupWithData(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	g, _ := c.Groups.Create(ctx, "성장주", 100.0)
	s, _ := c.Stocks.Create(ctx, "005930", g.ID)
	acc, _ := c.Accounts.Create(ctx, "내 계좌", numeric.Zero)
	_, _ = c.Holdings.Create(ctx, acc.ID, s.ID, numeric.FromInt(10))

	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, nil, nil)
	groups, err := ps.GetHoldingsByGroup(ctx)
	if err != nil {
		t.Fatalf("GetHoldingsByGroup: %v", err)
	}
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if groups[0].Group.Name != "성장주" {
		t.Errorf("group name = %q, want 성장주", groups[0].Group.Name)
	}
}

func TestGetPortfolioSummaryNoPriceService(t *testing.T) {
	c := newPortfolioContainer(t)
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, nil, nil)
	_, err := ps.GetPortfolioSummary(context.Background(), false)
	if !errors.Is(err, services.ErrNoPriceService) {
		t.Errorf("expected ErrNoPriceService, got %v", err)
	}
}

func TestGetPortfolioSummaryWithDBPrices(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	priceService := services.NewPriceService(c.StockPrices)
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, nil)

	summary, err := ps.GetPortfolioSummary(ctx, false)
	if err != nil {
		t.Fatalf("GetPortfolioSummary: %v", err)
	}
	if summary == nil {
		t.Fatal("summary is nil")
	}
}

func TestGetPortfolioSummaryWithData(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	g, _ := c.Groups.Create(ctx, "성장주", 50.0)
	s, _ := c.Stocks.Create(ctx, "005930", g.ID)
	acc, _ := c.Accounts.Create(ctx, "내 계좌", numeric.FromInt(1000000))
	_, _ = c.Holdings.Create(ctx, acc.ID, s.ID, numeric.FromInt(10))

	// Seed price into DB (PriceService is DB-only).
	today, _ := datex.ParseDate("2026-06-01")
	p, _ := numeric.FromString("74000")
	_, _ = c.StockPrices.Save(ctx, "005930", today, p, "KRW", "삼성전자", sql.NullString{})

	priceService := services.NewPriceService(c.StockPrices)
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, nil)

	summary, err := ps.GetPortfolioSummary(ctx, false)
	if err != nil {
		t.Fatalf("GetPortfolioSummary with data: %v", err)
	}
	if summary == nil {
		t.Fatal("summary is nil")
	}
	groupRows := services.ComputeGroupSummary(summary)
	if len(groupRows) == 0 {
		t.Error("ComputeGroupSummary returned empty rows for non-empty portfolio")
	}
}

func TestGetPortfolioSummaryBenchmarkReturns(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	g, _ := c.Groups.Create(ctx, "성장주", 100.0)
	s, _ := c.Stocks.Create(ctx, "005930", g.ID)
	acc, _ := c.Accounts.Create(ctx, "내 계좌", numeric.Zero)
	_, _ = c.Holdings.Create(ctx, acc.ID, s.ID, numeric.FromInt(120))

	start := datex.New(2026, time.January, 1)
	today := datex.New(2026, time.June, 1)
	_, _ = c.Deposits.Create(ctx, numeric.FromInt(100), start, sql.NullString{})

	savePrice := func(ticker, price, currency, name, exchange string, d datex.Date) {
		t.Helper()
		p, _ := numeric.FromString(price)
		ex := sql.NullString{}
		if exchange != "" {
			ex = sql.NullString{String: exchange, Valid: true}
		}
		if _, err := c.StockPrices.Save(ctx, ticker, d, p, currency, name, ex); err != nil {
			t.Fatalf("save price %s %s: %v", ticker, d.ISO(), err)
		}
	}

	savePrice("005930", "1", "KRW", "삼성전자", "", today)
	savePrice("SPY", "100", "USD", "SPDR S&P 500 ETF", "AMEX", start)
	savePrice("SPY", "110", "USD", "SPDR S&P 500 ETF", "AMEX", today)
	savePrice("QQQ", "100", "USD", "Invesco QQQ Trust", "NASD", start)
	savePrice("QQQ", "130", "USD", "Invesco QQQ Trust", "NASD", today)
	savePrice("226490", "100", "KRW", "KODEX KOSPI", "", start)
	savePrice("226490", "90", "KRW", "KODEX KOSPI", "", today)

	priceService := services.NewPriceService(c.StockPrices).WithTodayProvider(func() time.Time { return today.Time })
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, nil)

	summary, err := ps.GetPortfolioSummary(ctx, true)
	if err != nil {
		t.Fatalf("GetPortfolioSummary: %v", err)
	}
	if len(summary.BenchmarkReturns) != 3 {
		t.Fatalf("BenchmarkReturns len = %d, want 3", len(summary.BenchmarkReturns))
	}
	if summary.BenchmarkAverageReturn == nil {
		t.Fatal("BenchmarkAverageReturn is nil")
	}
	if got, want := summary.BenchmarkAverageReturn.StringFixed(1), "10.0"; got != want {
		t.Errorf("BenchmarkAverageReturn = %s, want %s", got, want)
	}
	if summary.BenchmarkAverageDiff == nil {
		t.Fatal("BenchmarkAverageDiff is nil")
	}
	if got, want := summary.BenchmarkAverageDiff.StringFixed(1), "10.0"; got != want {
		t.Errorf("BenchmarkAverageDiff = %s, want %s", got, want)
	}
}

// When the portfolio return is nil (e.g. deposits net to zero) but a first
// deposit date exists, benchmark return rates are still shown — only the
// per-benchmark Difference column is omitted.
func TestGetPortfolioSummaryBenchmarkReturnsWithoutPortfolioReturn(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	g, _ := c.Groups.Create(ctx, "성장주", 100.0)
	s, _ := c.Stocks.Create(ctx, "005930", g.ID)
	acc, _ := c.Accounts.Create(ctx, "내 계좌", numeric.Zero)
	_, _ = c.Holdings.Create(ctx, acc.ID, s.ID, numeric.FromInt(120))

	start := datex.New(2026, time.January, 1)
	today := datex.New(2026, time.June, 1)
	// Net-zero deposits: first-deposit date is set but totalInvested is 0, so the
	// portfolio return rate is nil.
	_, _ = c.Deposits.Create(ctx, numeric.FromInt(100), start, sql.NullString{})
	_, _ = c.Deposits.Create(ctx, numeric.FromInt(-100), datex.New(2026, time.March, 1), sql.NullString{})

	savePrice := func(ticker, price, currency, name, exchange string, d datex.Date) {
		t.Helper()
		p, _ := numeric.FromString(price)
		ex := sql.NullString{}
		if exchange != "" {
			ex = sql.NullString{String: exchange, Valid: true}
		}
		if _, err := c.StockPrices.Save(ctx, ticker, d, p, currency, name, ex); err != nil {
			t.Fatalf("save price %s %s: %v", ticker, d.ISO(), err)
		}
	}

	savePrice("005930", "1", "KRW", "삼성전자", "", today)
	savePrice("SPY", "100", "USD", "SPDR S&P 500 ETF", "AMEX", start)
	savePrice("SPY", "110", "USD", "SPDR S&P 500 ETF", "AMEX", today)
	savePrice("QQQ", "100", "USD", "Invesco QQQ Trust", "NASD", start)
	savePrice("QQQ", "130", "USD", "Invesco QQQ Trust", "NASD", today)
	savePrice("226490", "100", "KRW", "KODEX KOSPI", "", start)
	savePrice("226490", "90", "KRW", "KODEX KOSPI", "", today)

	priceService := services.NewPriceService(c.StockPrices).WithTodayProvider(func() time.Time { return today.Time })
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, nil)

	summary, err := ps.GetPortfolioSummary(ctx, true)
	if err != nil {
		t.Fatalf("GetPortfolioSummary: %v", err)
	}
	if summary.ReturnRate != nil {
		t.Fatalf("ReturnRate = %v, want nil (net-zero deposits)", summary.ReturnRate)
	}
	if len(summary.BenchmarkReturns) != 3 {
		t.Fatalf("BenchmarkReturns len = %d, want 3", len(summary.BenchmarkReturns))
	}
	for _, b := range summary.BenchmarkReturns {
		if b.ReturnRate == nil {
			t.Errorf("benchmark %s: ReturnRate is nil, want a rate", b.Ticker)
		}
		if b.Difference != nil {
			t.Errorf("benchmark %s: Difference = %v, want nil (no portfolio return)", b.Ticker, b.Difference)
		}
	}
	if summary.BenchmarkAvailableCount != 3 {
		t.Errorf("BenchmarkAvailableCount = %d, want 3", summary.BenchmarkAvailableCount)
	}
}

// BenchmarkAvailableCount reflects how many benchmarks have a usable return rate,
// so a partial-coverage average can be distinguished from full coverage.
func TestGetPortfolioSummaryBenchmarkAvailableCountPartial(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	g, _ := c.Groups.Create(ctx, "성장주", 100.0)
	s, _ := c.Stocks.Create(ctx, "005930", g.ID)
	acc, _ := c.Accounts.Create(ctx, "내 계좌", numeric.Zero)
	_, _ = c.Holdings.Create(ctx, acc.ID, s.ID, numeric.FromInt(120))

	start := datex.New(2026, time.January, 1)
	today := datex.New(2026, time.June, 1)
	_, _ = c.Deposits.Create(ctx, numeric.FromInt(100), start, sql.NullString{})

	savePrice := func(ticker, price, currency, name, exchange string, d datex.Date) {
		t.Helper()
		p, _ := numeric.FromString(price)
		ex := sql.NullString{}
		if exchange != "" {
			ex = sql.NullString{String: exchange, Valid: true}
		}
		if _, err := c.StockPrices.Save(ctx, ticker, d, p, currency, name, ex); err != nil {
			t.Fatalf("save price %s %s: %v", ticker, d.ISO(), err)
		}
	}

	savePrice("005930", "1", "KRW", "삼성전자", "", today)
	// Only SPY has both start and current prices; QQQ and 226490 have none, so
	// their change-since-start is nil.
	savePrice("SPY", "100", "USD", "SPDR S&P 500 ETF", "AMEX", start)
	savePrice("SPY", "110", "USD", "SPDR S&P 500 ETF", "AMEX", today)

	priceService := services.NewPriceService(c.StockPrices).WithTodayProvider(func() time.Time { return today.Time })
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, nil)

	summary, err := ps.GetPortfolioSummary(ctx, true)
	if err != nil {
		t.Fatalf("GetPortfolioSummary: %v", err)
	}
	if len(summary.BenchmarkReturns) != 3 {
		t.Fatalf("BenchmarkReturns len = %d, want 3", len(summary.BenchmarkReturns))
	}
	if summary.BenchmarkAvailableCount != 1 {
		t.Errorf("BenchmarkAvailableCount = %d, want 1 (only SPY priced)", summary.BenchmarkAvailableCount)
	}
}

func TestGetPortfolioSummaryUSDStock(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	g, _ := c.Groups.Create(ctx, "해외주", 50.0)
	exchange := "NASD"
	s, _ := c.Stocks.Create(ctx, "AAPL", g.ID)
	_, _ = c.Stocks.UpdateExchange(ctx, s.ID, exchange)
	s.Exchange = &exchange
	acc, _ := c.Accounts.Create(ctx, "해외계좌", numeric.Zero)
	_, _ = c.Holdings.Create(ctx, acc.ID, s.ID, numeric.FromInt(5))

	// Seed USD price into DB.
	today, _ := datex.ParseDate("2026-06-01")
	p, _ := numeric.FromString("195.89")
	_, _ = c.StockPrices.Save(ctx, "AAPL", today, p, "USD", "Apple Inc.", sql.NullString{String: "NASD", Valid: true})

	rate, _ := numeric.FromString("1300")
	exchangeRate := services.NewFixedExchangeRateService(rate)
	priceService := services.NewPriceService(c.StockPrices)
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, exchangeRate)

	summary, err := ps.GetPortfolioSummary(ctx, false)
	if err != nil {
		t.Fatalf("GetPortfolioSummary USD: %v", err)
	}
	if summary == nil {
		t.Fatal("summary is nil")
	}
}

type countingEximClient struct {
	calls int
	rate  float64
}

func (c *countingEximClient) FetchUSDRate(searchDate string) (float64, error) {
	c.calls++
	return c.rate, nil
}

func TestGetPortfolioSummaryKRWOnlyNoRateFetch(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	g, _ := c.Groups.Create(ctx, "국내주", 100.0)
	s, _ := c.Stocks.Create(ctx, "005930", g.ID)
	acc, _ := c.Accounts.Create(ctx, "내 계좌", numeric.Zero)
	_, _ = c.Holdings.Create(ctx, acc.ID, s.ID, numeric.FromInt(10))

	today, _ := datex.ParseDate("2026-06-01")
	p, _ := numeric.FromString("74000")
	_, _ = c.StockPrices.Save(ctx, "005930", today, p, "KRW", "삼성전자", sql.NullString{})

	exim := &countingEximClient{rate: 1300}
	exchangeRate := services.NewEximExchangeRateService(exim)
	priceService := services.NewPriceService(c.StockPrices)
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, exchangeRate)

	if _, err := ps.GetPortfolioSummary(ctx, false); err != nil {
		t.Fatalf("GetPortfolioSummary KRW-only: %v", err)
	}
	if exim.calls != 0 {
		t.Errorf("KRW-only portfolio fetched USD rate %d time(s), want 0", exim.calls)
	}
}

func TestGetPortfolioSummaryUSDFetchesRate(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	g, _ := c.Groups.Create(ctx, "해외주", 100.0)
	exchange := "NASD"
	acc, _ := c.Accounts.Create(ctx, "해외계좌", numeric.Zero)
	today, _ := datex.ParseDate("2026-06-01")

	// Two USD holdings: proves the rate is fetched once per portfolio (memoized),
	// not once per holding.
	for _, tk := range []struct {
		ticker, name, price string
	}{
		{"AAPL", "Apple Inc.", "195.89"},
		{"MSFT", "Microsoft Corp.", "430.50"},
	} {
		s, _ := c.Stocks.Create(ctx, tk.ticker, g.ID)
		_, _ = c.Stocks.UpdateExchange(ctx, s.ID, exchange)
		_, _ = c.Holdings.Create(ctx, acc.ID, s.ID, numeric.FromInt(5))
		p, _ := numeric.FromString(tk.price)
		_, _ = c.StockPrices.Save(ctx, tk.ticker, today, p, "USD", tk.name, sql.NullString{String: "NASD", Valid: true})
	}

	exim := &countingEximClient{rate: 1300}
	exchangeRate := services.NewEximExchangeRateService(exim)
	priceService := services.NewPriceService(c.StockPrices)
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, exchangeRate)

	summary, err := ps.GetPortfolioSummary(ctx, false)
	if err != nil {
		t.Fatalf("GetPortfolioSummary USD: %v", err)
	}
	if exim.calls != 1 {
		t.Errorf("USD portfolio fetched USD rate %d time(s) for 2 USD holdings, want exactly 1 (memoized)", exim.calls)
	}
	if summary.USDKRWRate == nil {
		t.Error("USDKRWRate is nil for USD portfolio, want populated")
	}
}

func TestResolveAndPersistNameWithPriceService(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	g, _ := c.Groups.Create(ctx, "테스트그룹", 100.0)
	stock, _ := c.Stocks.Create(ctx, "AAPL", g.ID)

	priceService := services.NewPriceService(c.StockPrices)
	ss := services.NewStockService(c.Stocks, priceService)
	result := ss.ResolveAndPersistName(ctx, &stock)
	// DB has no price/name data, result is ""
	_ = result
}

// Timing-matched mode replays each deposit at its own date, so a later deposit
// made after the benchmark already ran up earns only the remaining move — the
// whole point of the mode versus lump-sum.
func TestGetPortfolioSummaryTimingMatchedBenchmarks(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	g, _ := c.Groups.Create(ctx, "성장주", 100.0)
	s, _ := c.Stocks.Create(ctx, "005930", g.ID)
	acc, _ := c.Accounts.Create(ctx, "내 계좌", numeric.Zero)
	_, _ = c.Holdings.Create(ctx, acc.ID, s.ID, numeric.FromInt(300))

	start := datex.New(2026, time.January, 1)
	mid := datex.New(2026, time.March, 1)
	today := datex.New(2026, time.June, 1)
	_, _ = c.Deposits.Create(ctx, numeric.FromInt(100), start, sql.NullString{})
	_, _ = c.Deposits.Create(ctx, numeric.FromInt(100), mid, sql.NullString{})

	savePrice := func(ticker, price string, d datex.Date) {
		t.Helper()
		p, _ := numeric.FromString(price)
		if _, err := c.StockPrices.Save(ctx, ticker, d, p, "KRW", ticker, sql.NullString{}); err != nil {
			t.Fatalf("save price %s %s: %v", ticker, d.ISO(), err)
		}
	}

	savePrice("005930", "1", today)
	// 100 at 100 → 1 unit; 100 at 200 → 0.5 unit; 1.5 units × 200 = 300 on 200 invested = +50%.
	savePrice("360750", "100", start)
	savePrice("360750", "200", mid)
	savePrice("360750", "200", today)
	// 100 at 100 → 1 unit; 100 at 100 → 1 unit; 2 units × 100 = 200 on 200 = 0%.
	savePrice("368590", "100", start)
	savePrice("368590", "100", mid)
	savePrice("368590", "100", today)
	savePrice("226490", "100", start)
	savePrice("226490", "50", mid)
	savePrice("226490", "50", today)

	priceService := services.NewPriceService(c.StockPrices).WithTodayProvider(func() time.Time { return today.Time })
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, nil)

	summary, err := ps.GetPortfolioSummaryWithBenchmarkMode(ctx, false, services.BenchmarkModeTimingMatched)
	if err != nil {
		t.Fatalf("GetPortfolioSummaryWithBenchmarkMode: %v", err)
	}
	if summary.BenchmarkMode != string(services.BenchmarkModeTimingMatched) {
		t.Errorf("BenchmarkMode = %q, want timing-matched", summary.BenchmarkMode)
	}
	want := map[string]string{"360750": "50.0", "368590": "0.0", "226490": "-25.0"}
	if len(summary.BenchmarkReturns) != len(want) {
		t.Fatalf("BenchmarkReturns len = %d, want %d", len(summary.BenchmarkReturns), len(want))
	}
	for _, b := range summary.BenchmarkReturns {
		exp, ok := want[b.Ticker]
		if !ok {
			t.Errorf("unexpected benchmark ticker %q", b.Ticker)
			continue
		}
		if b.ReturnRate == nil {
			t.Errorf("%s ReturnRate is nil", b.Ticker)
			continue
		}
		if got := b.ReturnRate.StringFixed(1); got != exp {
			t.Errorf("%s ReturnRate = %s, want %s", b.Ticker, got, exp)
		}
	}
	// Portfolio: 300 assets on 200 invested = +50%, so the diff vs 360750 is 0.
	if summary.ReturnRate == nil || summary.ReturnRate.StringFixed(1) != "50.0" {
		t.Fatalf("ReturnRate = %v, want 50.0", summary.ReturnRate)
	}
	for _, b := range summary.BenchmarkReturns {
		if b.Ticker == "360750" {
			if b.Difference == nil || b.Difference.StringFixed(1) != "0.0" {
				t.Errorf("360750 Difference = %v, want 0.0", b.Difference)
			}
		}
	}
}

// A deposit predating every cached close makes the whole simulation unusable:
// dropping that deposit would shrink the invested base and overstate the
// benchmark, so the benchmark reports nil instead.
func TestGetPortfolioSummaryTimingMatchedSkipsUnpriceableBenchmark(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	g, _ := c.Groups.Create(ctx, "성장주", 100.0)
	s, _ := c.Stocks.Create(ctx, "005930", g.ID)
	acc, _ := c.Accounts.Create(ctx, "내 계좌", numeric.Zero)
	_, _ = c.Holdings.Create(ctx, acc.ID, s.ID, numeric.FromInt(300))

	early := datex.New(2020, time.January, 1)
	start := datex.New(2026, time.January, 1)
	today := datex.New(2026, time.June, 1)
	_, _ = c.Deposits.Create(ctx, numeric.FromInt(100), early, sql.NullString{})
	_, _ = c.Deposits.Create(ctx, numeric.FromInt(100), start, sql.NullString{})

	savePrice := func(ticker, price string, d datex.Date) {
		t.Helper()
		p, _ := numeric.FromString(price)
		if _, err := c.StockPrices.Save(ctx, ticker, d, p, "KRW", ticker, sql.NullString{}); err != nil {
			t.Fatalf("save price %s %s: %v", ticker, d.ISO(), err)
		}
	}

	savePrice("005930", "1", today)
	// Only 360750 reaches back before the 2020 deposit.
	savePrice("360750", "100", early)
	savePrice("360750", "100", start)
	savePrice("360750", "150", today)
	savePrice("368590", "100", start)
	savePrice("368590", "150", today)
	savePrice("226490", "100", start)
	savePrice("226490", "150", today)

	priceService := services.NewPriceService(c.StockPrices).WithTodayProvider(func() time.Time { return today.Time })
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, nil)

	summary, err := ps.GetPortfolioSummaryWithBenchmarkMode(ctx, false, services.BenchmarkModeTimingMatched)
	if err != nil {
		t.Fatalf("GetPortfolioSummaryWithBenchmarkMode: %v", err)
	}
	if summary.BenchmarkAvailableCount != 1 {
		t.Errorf("BenchmarkAvailableCount = %d, want 1 (only 360750 covers every deposit)", summary.BenchmarkAvailableCount)
	}
	for _, b := range summary.BenchmarkReturns {
		if b.Ticker == "360750" {
			if b.ReturnRate == nil || b.ReturnRate.StringFixed(1) != "50.0" {
				t.Errorf("360750 ReturnRate = %v, want 50.0", b.ReturnRate)
			}
			continue
		}
		if b.ReturnRate != nil {
			t.Errorf("%s ReturnRate = %v, want nil", b.Ticker, b.ReturnRate)
		}
	}
}

// The default path must keep reporting lump-sum numbers under a lump-sum label.
func TestGetPortfolioSummaryDefaultsToLumpSumMode(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	priceService := services.NewPriceService(c.StockPrices)
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, nil)

	summary, err := ps.GetPortfolioSummary(ctx, false)
	if err != nil {
		t.Fatalf("GetPortfolioSummary: %v", err)
	}
	if summary.BenchmarkMode != string(services.BenchmarkModeLumpSum) {
		t.Errorf("BenchmarkMode = %q, want lump-sum", summary.BenchmarkMode)
	}
}

func TestGetPortfolioSummaryRejectsUnknownBenchmarkMode(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	priceService := services.NewPriceService(c.StockPrices)
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, nil)

	if _, err := ps.GetPortfolioSummaryWithBenchmarkMode(ctx, false, services.BenchmarkMode("dollar-cost")); err == nil {
		t.Fatal("expected error for unknown benchmark mode, got nil")
	}
}

// A withdrawal must sell benchmark units, not merely shrink the invested base:
// keeping the units while the denominator drops fabricates benchmark return out
// of a correction. 100 buys 1 unit at 100; -50 sells 0.25 unit at 200; the
// remaining 0.75 unit is worth 150 on 50 invested = +200%, not the +300% that
// an unsold-units simulation would report.
func TestGetPortfolioSummaryTimingMatchedSellsUnitsForNegativeDeposit(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	g, _ := c.Groups.Create(ctx, "성장주", 100.0)
	s, _ := c.Stocks.Create(ctx, "005930", g.ID)
	acc, _ := c.Accounts.Create(ctx, "내 계좌", numeric.Zero)
	_, _ = c.Holdings.Create(ctx, acc.ID, s.ID, numeric.FromInt(100))

	start := datex.New(2026, time.January, 1)
	mid := datex.New(2026, time.March, 1)
	today := datex.New(2026, time.March, 20)
	_, _ = c.Deposits.Create(ctx, numeric.FromInt(100), start, sql.NullString{})
	_, _ = c.Deposits.Create(ctx, numeric.FromInt(-50), mid, sql.NullString{})

	savePrice := func(ticker, price string, d datex.Date) {
		t.Helper()
		p, _ := numeric.FromString(price)
		if _, err := c.StockPrices.Save(ctx, ticker, d, p, "KRW", ticker, sql.NullString{}); err != nil {
			t.Fatalf("save price %s %s: %v", ticker, d.ISO(), err)
		}
	}

	savePrice("005930", "1", today)
	for _, ticker := range []string{"360750", "368590", "226490"} {
		savePrice(ticker, "100", start)
		savePrice(ticker, "200", mid)
		savePrice(ticker, "200", today)
	}

	priceService := services.NewPriceService(c.StockPrices).WithTodayProvider(func() time.Time { return today.Time })
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, nil)

	summary, err := ps.GetPortfolioSummaryWithBenchmarkMode(ctx, false, services.BenchmarkModeTimingMatched)
	if err != nil {
		t.Fatalf("GetPortfolioSummaryWithBenchmarkMode: %v", err)
	}
	for _, b := range summary.BenchmarkReturns {
		if b.ReturnRate == nil {
			t.Fatalf("%s ReturnRate is nil", b.Ticker)
		}
		if got := b.ReturnRate.StringFixed(1); got != "200.0" {
			t.Errorf("%s ReturnRate = %s, want 200.0 (withdrawal sells units)", b.Ticker, got)
		}
	}
}

// Price history is sparse checkpoints, so an on-or-before lookup can resolve a
// deposit to a close from years earlier. Pricing the deposit off that close
// while labelling the result timing-matched is worse than reporting nothing.
func TestGetPortfolioSummaryTimingMatchedRejectsStaleDepositPrice(t *testing.T) {
	c := newPortfolioContainer(t)
	ctx := context.Background()

	g, _ := c.Groups.Create(ctx, "성장주", 100.0)
	s, _ := c.Stocks.Create(ctx, "005930", g.ID)
	acc, _ := c.Accounts.Create(ctx, "내 계좌", numeric.Zero)
	_, _ = c.Holdings.Create(ctx, acc.ID, s.ID, numeric.FromInt(100))

	old := datex.New(2021, time.January, 6)
	deposit := datex.New(2023, time.June, 1)
	today := datex.New(2026, time.June, 1)
	_, _ = c.Deposits.Create(ctx, numeric.FromInt(100), deposit, sql.NullString{})

	savePrice := func(ticker, price string, d datex.Date) {
		t.Helper()
		p, _ := numeric.FromString(price)
		if _, err := c.StockPrices.Save(ctx, ticker, d, p, "KRW", ticker, sql.NullString{}); err != nil {
			t.Fatalf("save price %s %s: %v", ticker, d.ISO(), err)
		}
	}

	savePrice("005930", "1", today)
	// A 2021 close and a current close, with the 2023 deposit falling in the hole
	// between them — exactly the shape `pm price-sync` leaves behind.
	for _, ticker := range []string{"360750", "368590", "226490"} {
		savePrice(ticker, "100", old)
		savePrice(ticker, "200", today)
	}

	priceService := services.NewPriceService(c.StockPrices).WithTodayProvider(func() time.Time { return today.Time })
	ps := services.NewPortfolioService(c.Groups, c.Stocks, c.Holdings, c.Accounts, c.Deposits, priceService, nil)

	summary, err := ps.GetPortfolioSummaryWithBenchmarkMode(ctx, false, services.BenchmarkModeTimingMatched)
	if err != nil {
		t.Fatalf("GetPortfolioSummaryWithBenchmarkMode: %v", err)
	}
	if summary.BenchmarkAvailableCount != 0 {
		t.Errorf("BenchmarkAvailableCount = %d, want 0 (every deposit price is stale)", summary.BenchmarkAvailableCount)
	}
	for _, b := range summary.BenchmarkReturns {
		if b.ReturnRate != nil {
			t.Errorf("%s ReturnRate = %v, want nil (priced off a 2021 close)", b.Ticker, b.ReturnRate)
		}
	}
}
