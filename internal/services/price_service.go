package services

import (
	"context"
	"time"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/repositories"
)

// supportedPeriods is the set of valid change-rate period labels.
var supportedPeriods = map[string]bool{"1d": true, "1m": true, "6m": true, "1y": true}

// orderToPrice maps order-form exchange codes to price-form codes (KIS convention).
var orderToPrice = map[string]string{
	"NAS": "NASD",
	"NYS": "NYSE",
	"AMS": "AMEX",
}

// priceToOrder is the reverse mapping.
var priceToOrder = map[string]string{
	"NASD": "NAS",
	"NYSE": "NYS",
	"AMEX": "AMS",
}

// PriceService resolves stock prices from the DB cache only.
// PriceSyncService owns live API access and keeps the DB up to date.
type PriceService struct {
	stockPrices   *repositories.StockPriceRepository
	todayProvider func() time.Time
}

// NewPriceService creates a DB-only PriceService. Use PriceSyncService for live fetching.
func NewPriceService(stockPrices *repositories.StockPriceRepository) *PriceService {
	return &PriceService{
		stockPrices:   stockPrices,
		todayProvider: func() time.Time { return ktime.Now().Time },
	}
}

// WithTodayProvider overrides the service's date source. Use in tests to fix today's date.
func (s *PriceService) WithTodayProvider(fn func() time.Time) *PriceService {
	s.todayProvider = fn
	return s
}

// GetStockPrice returns (price, currency, name, exchange) for ticker from DB.
// Returns today's price if available, otherwise the most recent cached price.
// Returns zero when no data exists.
func (s *PriceService) GetStockPrice(ctx context.Context, ticker, preferredExchange string) (numeric.Decimal, string, string, string) {
	orderExch := toOrderExchange(preferredExchange)
	today := datex.FromTime(s.todayProvider())
	if sp := s.loadCached(ctx, ticker, today); sp != nil {
		return sp.Price, sp.Currency, sp.Name, toOrderExchange(sp.Exchange.String)
	}
	return numeric.Zero, "KRW", "", orderExch
}

// GetCachedPrice returns the stored price for (ticker, date) from the DB, or nil.
func (s *PriceService) GetCachedPrice(ctx context.Context, ticker string, date datex.Date) *models.StockPrice {
	return s.loadCached(ctx, ticker, date)
}

// loadCached returns today's price or, if absent, the most-recent stored price (stale fallback).
// Use getExact when the fallback would give misleading results (e.g. change-rate history).
func (s *PriceService) loadCached(ctx context.Context, ticker string, date datex.Date) *models.StockPrice {
	if s.stockPrices == nil {
		return nil
	}
	sp, _ := s.stockPrices.GetByTickerAndDate(ctx, ticker, date)
	if sp != nil && sp.Price.IsPositive() {
		return sp
	}
	// Fall back to most-recent available price (handles weekends and holidays).
	sp, _ = s.stockPrices.GetLatestByTicker(ctx, ticker)
	if sp != nil && sp.Price.IsPositive() {
		return sp
	}
	return nil
}

// getOnOrBefore returns the most recent stored price at or before the given date,
// or nil. Unlike loadCached it never jumps forward to the latest price, so a
// non-business target date resolves to the nearest prior trading day rather than
// today's value — correct for historical change-rate lookups.
func (s *PriceService) getOnOrBefore(ctx context.Context, ticker string, date datex.Date) *models.StockPrice {
	if s.stockPrices == nil {
		return nil
	}
	sp, _ := s.stockPrices.GetOnOrBeforeDate(ctx, ticker, date)
	if sp != nil && sp.Price.IsPositive() {
		return sp
	}
	return nil
}

// GetStockChangeRates returns rate-of-change (%) for each period from DB.
// Returns nil when no current price is available or no valid periods requested.
func (s *PriceService) GetStockChangeRates(ctx context.Context, ticker, preferredExchange string, periods []string) map[string]numeric.Decimal {
	if s.stockPrices == nil {
		return nil
	}

	normalized := make([]string, 0, len(periods))
	seen := map[string]bool{}
	for _, p := range periods {
		if supportedPeriods[p] && !seen[p] {
			normalized = append(normalized, p)
			seen[p] = true
		}
	}
	if len(normalized) == 0 {
		return nil
	}

	currentPrice, _, _, _ := s.GetStockPrice(ctx, ticker, preferredExchange)
	if currentPrice.IsZero() {
		return nil
	}

	today := datex.FromTime(s.todayProvider())
	targetDates := computeTargetDates(today.Time)
	result := make(map[string]numeric.Decimal, len(normalized))

	for _, label := range normalized {
		target := targetDates[label]
		targetDate := datex.FromTime(target)

		var pastClose numeric.Decimal
		if cached := s.getOnOrBefore(ctx, ticker, targetDate); cached != nil && cached.Price.IsPositive() {
			pastClose = cached.Price
		}

		if pastClose.IsZero() {
			result[label] = numeric.Zero
			continue
		}
		rate := numeric.Wrap(
			currentPrice.Sub(pastClose.Decimal).Div(pastClose.Decimal).Mul(hundred.Decimal),
		)
		result[label] = rate
	}
	return result
}

// GetStockChangeSince returns rate-of-change (%) from the nearest cached close
// at or before startDate to the current cached price.
func (s *PriceService) GetStockChangeSince(ctx context.Context, ticker string, startDate datex.Date) *numeric.Decimal {
	if s.stockPrices == nil || startDate.Time.IsZero() {
		return nil
	}
	currentPrice, _, _, _ := s.GetStockPrice(ctx, ticker, "")
	if currentPrice.IsZero() {
		return nil
	}
	start := s.getOnOrBefore(ctx, ticker, startDate)
	if start == nil || !start.Price.IsPositive() {
		return nil
	}
	rate := numeric.Wrap(
		currentPrice.Sub(start.Price.Decimal).Div(start.Price.Decimal).Mul(hundred.Decimal),
	)
	return &rate
}

// computeTargetDates builds the historical reference date for each period.
func computeTargetDates(today time.Time) map[string]time.Time {
	return map[string]time.Time{
		"1y": prevBizDay(shiftYears(today, 1)),
		"6m": prevBizDay(shiftMonths(today, 6)),
		"1m": prevBizDay(shiftMonths(today, 1)),
		"1d": prevBizDay(today.AddDate(0, 0, -1)),
	}
}

func shiftYears(t time.Time, years int) time.Time {
	target := t.AddDate(-years, 0, 0)
	y, m, _ := target.Date()
	last := lastDayOfMonth(y, m)
	d := t.Day()
	if d > last {
		d = last
	}
	return time.Date(y, m, d, 0, 0, 0, 0, target.Location())
}

func shiftMonths(t time.Time, months int) time.Time {
	target := t.AddDate(0, -months, 0)
	y, m, _ := target.Date()
	last := lastDayOfMonth(y, m)
	d := t.Day()
	if d > last {
		d = last
	}
	return time.Date(y, m, d, 0, 0, 0, 0, target.Location())
}

func lastDayOfMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func prevBizDay(t time.Time) time.Time {
	switch t.Weekday() {
	case time.Saturday:
		return t.AddDate(0, 0, -1)
	case time.Sunday:
		return t.AddDate(0, 0, -2)
	default:
		return t
	}
}

func toOrderExchange(e string) string {
	if v, ok := priceToOrder[e]; ok {
		return v
	}
	return e
}

func toPriceExchange(e string) string {
	if v, ok := orderToPrice[e]; ok {
		return v
	}
	return e
}
