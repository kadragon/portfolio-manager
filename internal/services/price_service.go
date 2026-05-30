package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/repositories"
)

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
	if e, ok := s.priceCache[k]; ok {
		return e.price, e.currency, e.name, e.exchange
	}

	today := datex.FromTime(s.todayProvider())
	if sp := s.loadCached(ctx, ticker, today); sp != nil {
		e := priceCacheEntry{
			price:    sp.Price,
			currency: sp.Currency,
			name:     sp.Name,
			exchange: toOrderExchange(sp.Exchange.String),
		}
		s.priceCache[k] = e
		return e.price, e.currency, e.name, e.exchange
	}

	if s.client == nil {
		return numeric.Zero, "KRW", "", cacheExch
	}

	quote, err := s.client.GetPrice(ticker, toPriceExchange(cacheExch))
	if err != nil || quote.Price <= 0 {
		return numeric.Zero, "KRW", "", cacheExch
	}

	price, _ := numeric.FromString(fmt.Sprintf("%g", quote.Price))
	normalized := toOrderExchange(quote.Exchange)
	if price.IsPositive() {
		exc := sql.NullString{}
		if normalized != "" {
			exc = sql.NullString{String: normalized, Valid: true}
		}
		_, _ = s.stockPrices.Save(ctx, ticker, today, price, quote.Currency, quote.Name, exc)
	}

	e := priceCacheEntry{price: price, currency: quote.Currency, name: quote.Name, exchange: normalized}
	if price.IsPositive() {
		s.priceCache[k] = e
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
