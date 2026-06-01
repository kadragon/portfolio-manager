package services

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
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

// PriceService resolves stock prices using DB cache first, then a live PriceClient.
// When client is nil, only the DB cache is consulted (returns zero on cache miss).
type PriceService struct {
	mu            sync.RWMutex
	stockPrices   *repositories.StockPriceRepository
	client        PriceClient
	todayProvider func() time.Time
	priceCache    map[priceCacheKey]priceCacheEntry
}

type priceCacheKey struct {
	ticker   string
	exchange string
}

type priceCacheEntry struct {
	price    numeric.Decimal
	currency string
	name     string
	exchange string // order-form code
}

// NewPriceService creates a PriceService. client may be nil (DB-cache-only mode).
func NewPriceService(stockPrices *repositories.StockPriceRepository, client PriceClient) *PriceService {
	return &PriceService{
		stockPrices:   stockPrices,
		client:        client,
		todayProvider: func() time.Time { return ktime.Now().Time },
		priceCache:    make(map[priceCacheKey]priceCacheEntry),
	}
}

// GetStockPrice returns (price, currency, name, exchange) for ticker.
// preferredExchange is the order-form code ("NAS","NYSE",…) or empty for domestic.
// Returns zero price when no data is available and no client is configured.
func (s *PriceService) GetStockPrice(ctx context.Context, ticker, preferredExchange string) (numeric.Decimal, string, string, string) {
	cacheExch := toOrderExchange(preferredExchange)
	k := priceCacheKey{ticker: ticker, exchange: cacheExch}
	s.mu.RLock()
	if e, ok := s.priceCache[k]; ok {
		s.mu.RUnlock()
		return e.price, e.currency, e.name, e.exchange
	}
	s.mu.RUnlock()

	today := datex.FromTime(s.todayProvider())
	if sp := s.loadCached(ctx, ticker, today); sp != nil {
		e := priceCacheEntry{
			price:    sp.Price,
			currency: sp.Currency,
			name:     sp.Name,
			exchange: toOrderExchange(sp.Exchange.String),
		}
		s.mu.Lock()
		s.priceCache[k] = e
		s.mu.Unlock()
		return e.price, e.currency, e.name, e.exchange
	}

	if s.client == nil {
		return numeric.Zero, "KRW", "", cacheExch
	}

	quote, err := s.client.GetPrice(ticker, toPriceExchange(cacheExch))
	if err != nil || quote.Price <= 0 {
		if s.stockPrices != nil {
			if sp, _ := s.stockPrices.GetLatestByTicker(ctx, ticker); sp != nil && sp.Price.IsPositive() {
				e := priceCacheEntry{
					price:    sp.Price,
					currency: sp.Currency,
					name:     sp.Name,
					exchange: toOrderExchange(sp.Exchange.String),
				}
				s.mu.Lock()
				s.priceCache[k] = e
				s.mu.Unlock()
				return e.price, e.currency, e.name, e.exchange
			}
		}
		return numeric.Zero, "KRW", "", cacheExch
	}

	price, _ := numeric.FromString(fmt.Sprintf("%g", quote.Price))
	normalized := toOrderExchange(quote.Exchange)
	if price.IsPositive() {
		exc := sql.NullString{}
		if normalized != "" {
			exc = sql.NullString{String: normalized, Valid: true}
		}
		if s.stockPrices != nil {
			_, _ = s.stockPrices.Save(ctx, ticker, today, price, quote.Currency, quote.Name, exc)
		}
	}

	e := priceCacheEntry{price: price, currency: quote.Currency, name: quote.Name, exchange: normalized}
	if price.IsPositive() {
		s.mu.Lock()
		s.priceCache[k] = e
		s.mu.Unlock()
	}
	return e.price, e.currency, e.name, e.exchange
}

// GetCachedPrice returns the stored price for (ticker, date) from the DB, or nil.
func (s *PriceService) GetCachedPrice(ctx context.Context, ticker string, date datex.Date) *models.StockPrice {
	return s.loadCached(ctx, ticker, date)
}

func (s *PriceService) loadCached(ctx context.Context, ticker string, date datex.Date) *models.StockPrice {
	if s.stockPrices == nil {
		return nil
	}
	// Try today's price first (exact match like Python).
	sp, _ := s.stockPrices.GetByTickerAndDate(ctx, ticker, date)
	if sp != nil && sp.Price.IsPositive() {
		return sp
	}
	// Fall back to most-recent cached price when no KIS client is configured.
	// This lets the DB-only dashboard show holdings even when prices are from yesterday.
	if s.client == nil {
		sp, _ = s.stockPrices.GetLatestByTicker(ctx, ticker)
		if sp != nil && sp.Price.IsPositive() {
			return sp
		}
	}
	return nil
}

// GetStockChangeRates returns rate-of-change (%) for each period in periods.
// periods are labels from {"1d", "1m", "6m", "1y"}; invalid labels are ignored.
// Returns nil when no live client is available.
func (s *PriceService) GetStockChangeRates(ctx context.Context, ticker, preferredExchange string, periods []string) map[string]numeric.Decimal {
	if s.client == nil {
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

	currentPrice, currency, name, resolvedExchange := s.GetStockPrice(ctx, ticker, preferredExchange)
	priceExchange := toPriceExchange(resolvedExchange)
	today := datex.FromTime(s.todayProvider())

	targetDates := computeTargetDates(today.Time)
	result := make(map[string]numeric.Decimal, len(normalized))

	for _, label := range normalized {
		target := targetDates[label]
		targetDate := datex.FromTime(target)

		var pastClose numeric.Decimal
		if cached := s.loadCached(ctx, ticker, targetDate); cached != nil && cached.Price.IsPositive() {
			pastClose = cached.Price
		} else {
			raw, err := s.client.GetHistoricalClose(ticker, targetDate, priceExchange)
			if err == nil && raw > 0 {
				pastClose, _ = numeric.FromString(fmt.Sprintf("%g", raw))
				exc := sql.NullString{}
				if resolvedExchange != "" {
					exc = sql.NullString{String: resolvedExchange, Valid: true}
				}
				_, _ = s.stockPrices.Save(ctx, ticker, targetDate, pastClose, currency, name, exc)
			}
		}

		if pastClose.IsZero() {
			result[label] = numeric.Zero
			continue
		}
		// rate = (current - past) / past * 100
		rate := numeric.Wrap(
			currentPrice.Sub(pastClose.Decimal).Div(pastClose.Decimal).Mul(hundred.Decimal),
		)
		result[label] = rate
	}
	return result
}

// computeTargetDates builds the historical reference date for each period,
// mirroring Python's shift_years / shift_months / adjust_to_previous_business_day.
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
	// Clamp to last day of month if original day exceeds it.
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
	// First day of next month minus one day.
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
