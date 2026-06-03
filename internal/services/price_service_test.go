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

// TestGetStockPriceFromDB verifies that a saved price is returned directly.
func TestGetStockPriceFromDB(t *testing.T) {
	r := newPriceRepo(t)
	ctx := context.Background()

	today, _ := datex.ParseDate("2026-06-01")
	p, _ := numeric.FromString("74000")
	_, err := r.Save(ctx, "005930", today, p, "KRW", "삼성전자", sql.NullString{})
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	svc := services.NewPriceService(r)
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
}

// TestGetStockPriceConcurrentAccess verifies no data race under concurrent reads.
func TestGetStockPriceConcurrentAccess(t *testing.T) {
	r := newPriceRepo(t)
	ctx := context.Background()

	today, _ := datex.ParseDate("2026-06-01")
	p, _ := numeric.FromString("74000")
	_, _ = r.Save(ctx, "005930", today, p, "KRW", "삼성전자", sql.NullString{})

	svc := services.NewPriceService(r)
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

// TestGetStockPriceNoData verifies zero price when DB is empty.
func TestGetStockPriceNoData(t *testing.T) {
	r := newPriceRepo(t)
	svc := services.NewPriceService(r)
	ctx := context.Background()

	price, _, _, _ := svc.GetStockPrice(ctx, "005930", "")
	if !price.IsZero() {
		t.Errorf("want zero price with empty cache, got %v", price)
	}
}

// TestGetStockPriceStaleReturn verifies that a stale (non-today) price is returned
// when no today entry exists (e.g. weekend, market holiday).
func TestGetStockPriceStaleReturn(t *testing.T) {
	r := newPriceRepo(t)
	ctx := context.Background()

	staleDate, _ := datex.ParseDate("2026-05-29")
	cachedPrice, _ := numeric.FromString("159.11")
	_, err := r.Save(ctx, "VYM", staleDate, cachedPrice, "USD", "VANGUARD HIGH DIVIDEND YIELD",
		sql.NullString{String: "AMEX", Valid: true})
	if err != nil {
		t.Fatalf("save: %v", err)
	}

	svc := services.NewPriceService(r)
	price, currency, _, _ := svc.GetStockPrice(ctx, "VYM", "AMEX")
	if price.IsZero() {
		t.Error("want stale price from cache, got zero")
	}
	if !price.Equal(cachedPrice.Decimal) {
		t.Errorf("want cached price %v, got %v", cachedPrice, price)
	}
	if currency != "USD" {
		t.Errorf("want USD, got %s", currency)
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

	svc := services.NewPriceService(r)
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

// TestGetStockChangeRatesNoData returns nil when DB has no price data.
func TestGetStockChangeRatesNoData(t *testing.T) {
	r := newPriceRepo(t)
	svc := services.NewPriceService(r)
	ctx := context.Background()

	result := svc.GetStockChangeRates(ctx, "005930", "", []string{"1d"})
	if result != nil {
		t.Errorf("want nil result with no data, got %v", result)
	}
}

func TestGetStockChangeRatesEmptyPeriods(t *testing.T) {
	r := newPriceRepo(t)
	svc := services.NewPriceService(r)
	ctx := context.Background()

	result := svc.GetStockChangeRates(ctx, "005930", "", []string{})
	if result != nil {
		t.Errorf("want nil for empty periods, got %v", result)
	}
}

// TestGetStockChangeRatesFromDB verifies change rates are computed from DB data.
func TestGetStockChangeRatesFromDB(t *testing.T) {
	r := newPriceRepo(t)
	ctx := context.Background()

	// Pin today so computeTargetDates is deterministic.
	fixedToday := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	// Save current and 1-year-ago prices to DB.
	todayDate, _ := datex.ParseDate("2026-06-01")
	currentP, _ := numeric.FromString("100")
	_, _ = r.Save(ctx, "005930", todayDate, currentP, "KRW", "삼성전자", sql.NullString{})

	// prevBizDay(shiftYears(2026-06-01, 1)) = prevBizDay(2025-06-01 Sunday) = 2025-05-30
	pastDate, _ := datex.ParseDate("2025-05-30")
	pastP, _ := numeric.FromString("80")
	_, _ = r.Save(ctx, "005930", pastDate, pastP, "KRW", "삼성전자", sql.NullString{})

	svc := services.NewPriceService(r).WithTodayProvider(func() time.Time { return fixedToday })
	result := svc.GetStockChangeRates(ctx, "005930", "", []string{"1y"})
	if result == nil {
		t.Fatal("want non-nil result, got nil")
	}
	if _, ok := result["1y"]; !ok {
		t.Error("want '1y' key in result")
	}
	// rate = (100 - 80) / 80 * 100 = 25%
	if result["1y"].IsZero() {
		t.Error("want non-zero 1y rate, got zero")
	}
}

// TestGetStockChangeRatesZeroForMissingHistory returns a map with zero rates
// when current price exists but historical data is missing.
func TestGetStockChangeRatesZeroForMissingHistory(t *testing.T) {
	r := newPriceRepo(t)
	ctx := context.Background()

	todayDate, _ := datex.ParseDate("2026-06-01")
	p, _ := numeric.FromString("100")
	_, _ = r.Save(ctx, "005930", todayDate, p, "KRW", "삼성전자", sql.NullString{})

	svc := services.NewPriceService(r)
	result := svc.GetStockChangeRates(ctx, "005930", "", []string{"1y", "1m"})
	if result == nil {
		t.Fatal("want non-nil map (even with zeros), got nil")
	}
	if result["1y"].String() != "0" {
		t.Errorf("want zero 1y rate for missing history, got %v", result["1y"])
	}
}

// TestGetStockChangeRatesNearestPastDate verifies the rate is computed against
// the most recent price at or before the target date when no exact-date row
// exists (target lands on a non-business day with no cached price).
func TestGetStockChangeRatesNearestPastDate(t *testing.T) {
	r := newPriceRepo(t)
	ctx := context.Background()

	fixedToday := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	todayDate, _ := datex.ParseDate("2026-06-01")
	currentP, _ := numeric.FromString("100")
	_, _ = r.Save(ctx, "005930", todayDate, currentP, "KRW", "삼성전자", sql.NullString{})

	// 1y target = prevBizDay(2025-06-01 Sun) = 2025-05-30. No row exists there;
	// the nearest prior available price is 2025-05-28.
	pastDate, _ := datex.ParseDate("2025-05-28")
	pastP, _ := numeric.FromString("80")
	_, _ = r.Save(ctx, "005930", pastDate, pastP, "KRW", "삼성전자", sql.NullString{})

	svc := services.NewPriceService(r).WithTodayProvider(func() time.Time { return fixedToday })
	result := svc.GetStockChangeRates(ctx, "005930", "", []string{"1y"})
	if result == nil {
		t.Fatal("want non-nil result, got nil")
	}
	// rate = (100 - 80) / 80 * 100 = 25%; non-zero proves nearest-prior lookup.
	if result["1y"].IsZero() {
		t.Error("want non-zero 1y rate from nearest prior date, got zero")
	}
}
