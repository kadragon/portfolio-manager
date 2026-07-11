package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/repositories"
)

const syncCallDelay = 200 * time.Millisecond

// PriceSyncService fetches prices for all stocks and saves them to DB, one
// pass per SyncOnce call (invoked on demand via `pm price-sync`, not on a
// background schedule). It is the only component that calls PriceClient.
// PriceService reads from DB only.
type PriceSyncService struct {
	client      PriceClient
	stockPrices *repositories.StockPriceRepository
	stocks      *repositories.StockRepository
	deposits    *repositories.DepositRepository
}

// NewPriceSyncService constructs the service. All deps are required.
func NewPriceSyncService(
	client PriceClient,
	stockPrices *repositories.StockPriceRepository,
	stocks *repositories.StockRepository,
	deposits *repositories.DepositRepository,
) *PriceSyncService {
	return &PriceSyncService{
		client:      client,
		stockPrices: stockPrices,
		stocks:      stocks,
		deposits:    deposits,
	}
}

// SyncOnce fetches current prices for all stocks, then fills any missing
// historical closes (1y/6m/1m/1d). Historical closes are never re-fetched
// once saved — past data is immutable.
func (s *PriceSyncService) SyncOnce(ctx context.Context) {
	allStocks, err := s.stocks.ListAll(ctx)
	if err != nil {
		log.Printf("price sync: list stocks: %v", err)
		return
	}
	targets := syncTargets(allStocks)

	today := datex.FromTime(ktime.NowKST())
	historicalDates := s.syncHistoricalDates(ctx, today)

	for idx, target := range targets {
		if ctx.Err() != nil {
			return
		}

		preferredExchange := target.preferredExchange

		if idx > 0 {
			delay(ctx)
		}
		quote, err := s.client.GetPrice(target.ticker, preferredExchange)
		if (err != nil || quote.Price <= 0) && ctx.Err() == nil {
			// A single transient failure/empty response would otherwise leave this
			// ticker stuck on its last saved price for the whole sync pass.
			delay(ctx)
			quote, err = s.client.GetPrice(target.ticker, preferredExchange)
		}
		if err != nil {
			log.Printf("price sync: get price %s: %v", target.ticker, err)
			continue
		}
		if quote.Price <= 0 {
			continue
		}

		price, parseErr := numeric.FromString(fmt.Sprintf("%g", quote.Price))
		if parseErr != nil || !price.IsPositive() {
			continue
		}

		exc := sql.NullString{}
		if quote.Exchange != "" {
			exc = sql.NullString{String: toOrderExchange(quote.Exchange), Valid: true}
		}
		if _, err := s.stockPrices.Save(ctx, target.ticker, today, price, quote.Currency, quote.Name, exc); err != nil {
			log.Printf("price sync: save %s: %v", target.ticker, err)
		}

		// Persist resolved exchange (canonical form) to stock when it differs from the stored value.
		// canonical is the 4-letter code (NASD/NYSE/AMEX) expected by GetPrice/prioritizedExchanges.
		if target.stock != nil && quote.Exchange != "" {
			canonical := quote.Exchange
			if target.stock.Exchange == nil || *target.stock.Exchange != canonical {
				if _, err := s.stocks.UpdateExchange(ctx, target.stock.ID, canonical); err == nil {
					target.stock.Exchange = &canonical
					preferredExchange = canonical
				} else {
					log.Printf("price sync: update exchange %s: %v", target.ticker, err)
				}
			}
		}

		// Fetch any missing historical closes (fetch-once: immutable past data).
		for _, targetDate := range historicalDates {
			if cached, _ := s.stockPrices.GetByTickerAndDate(ctx, target.ticker, targetDate); cached != nil && cached.Price.IsPositive() {
				continue
			}
			delay(ctx)
			if ctx.Err() != nil {
				return
			}
			raw, histErr := s.client.GetHistoricalClose(target.ticker, targetDate, preferredExchange)
			if histErr != nil || raw <= 0 {
				continue
			}
			pastClose, parseErr := numeric.FromString(fmt.Sprintf("%g", raw))
			if parseErr != nil || !pastClose.IsPositive() {
				continue
			}
			if _, err := s.stockPrices.Save(ctx, target.ticker, targetDate, pastClose, quote.Currency, quote.Name, exc); err != nil {
				log.Printf("price sync: save historical %s: %v", target.ticker, err)
			}
		}
	}
}

// backfillWindowDays keeps a single GetHistoricalRange call under KIS's ~100-row
// period-endpoint cap; larger requested ranges are chunked into successive calls.
const backfillWindowDays = 90

// BackfillRangeResult reports what a BackfillRange call did for one ticker.
type BackfillRangeResult struct {
	Ticker    string
	Requested int
	Saved     int
	Skipped   int
}

// BackfillRange fetches and stores every daily close available for ticker within
// [start, end] that isn't already cached. Unlike SyncOnce's fixed 1y/6m/1m/1d
// checkpoints, this is for on-demand arbitrary-range backfills (e.g. "every day
// this month"). Existing rows are never overwritten (past data is immutable).
func (s *PriceSyncService) BackfillRange(ctx context.Context, ticker string, start, end datex.Date) (BackfillRangeResult, error) {
	result := BackfillRangeResult{Ticker: ticker}
	if end.ISO() < start.ISO() {
		return result, fmt.Errorf("backfill range: end %s before start %s", end.ISO(), start.ISO())
	}

	stock, err := s.stocks.GetByTicker(ctx, ticker)
	if err != nil {
		return result, err
	}
	preferredExchange := ""
	if stock != nil && stock.Exchange != nil {
		preferredExchange = *stock.Exchange
	}

	quote, err := s.client.GetPrice(ticker, preferredExchange)
	if err != nil {
		return result, fmt.Errorf("get current quote for %s: %w", ticker, err)
	}
	exc := sql.NullString{}
	if quote.Exchange != "" {
		exc = sql.NullString{String: toOrderExchange(quote.Exchange), Valid: true}
		preferredExchange = quote.Exchange
	}

	windowStart := start
	for windowStart.ISO() <= end.ISO() {
		delay(ctx)
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		windowEnd := datex.FromTime(windowStart.Time.AddDate(0, 0, backfillWindowDays))
		if windowEnd.ISO() > end.ISO() {
			windowEnd = end
		}

		points, rangeErr := s.client.GetHistoricalRange(ticker, windowStart, windowEnd, preferredExchange)
		if rangeErr != nil {
			return result, fmt.Errorf("backfill range %s %s..%s: %w", ticker, windowStart.ISO(), windowEnd.ISO(), rangeErr)
		}
		result.Requested += len(points)

		for _, p := range points {
			if p.Date.ISO() < start.ISO() || p.Date.ISO() > end.ISO() {
				continue
			}
			if cached, _ := s.stockPrices.GetByTickerAndDate(ctx, ticker, p.Date); cached != nil && cached.Price.IsPositive() {
				result.Skipped++
				continue
			}
			price, parseErr := numeric.FromString(fmt.Sprintf("%g", p.Price))
			if parseErr != nil || !price.IsPositive() {
				continue
			}
			if _, saveErr := s.stockPrices.Save(ctx, ticker, p.Date, price, quote.Currency, quote.Name, exc); saveErr != nil {
				log.Printf("price backfill: save %s %s: %v", ticker, p.Date.ISO(), saveErr)
				continue
			}
			result.Saved++
		}

		windowStart = datex.FromTime(windowEnd.Time.AddDate(0, 0, 1))
	}

	return result, nil
}

func (s *PriceSyncService) syncHistoricalDates(ctx context.Context, today datex.Date) []datex.Date {
	targetDates := computeTargetDates(today.Time)
	dates := make([]datex.Date, 0, len(targetDates)+1)
	seen := make(map[string]bool, len(targetDates)+1)
	for _, label := range []string{"1y", "6m", "1m", "1d"} {
		d := datex.FromTime(targetDates[label])
		dates = append(dates, d)
		seen[d.ISO()] = true
	}
	if s.deposits == nil {
		return dates
	}
	firstDate, err := s.deposits.GetFirstDepositDate(ctx)
	if err != nil || firstDate == nil || firstDate.Time.IsZero() {
		return dates
	}
	if firstDate.ISO() >= today.ISO() {
		return dates
	}
	adjusted := datex.FromTime(prevBizDay(firstDate.Time))
	if seen[adjusted.ISO()] {
		return dates
	}
	return append(dates, adjusted)
}

type priceSyncTarget struct {
	ticker            string
	preferredExchange string
	stock             *models.Stock
}

func syncTargets(stocks []models.Stock) []priceSyncTarget {
	targets := make([]priceSyncTarget, 0, len(stocks)+len(dashboardBenchmarks))
	seen := make(map[string]bool, len(stocks)+len(dashboardBenchmarks))
	for i := range stocks {
		preferredExchange := ""
		if stocks[i].Exchange != nil {
			preferredExchange = *stocks[i].Exchange
		}
		targets = append(targets, priceSyncTarget{
			ticker:            stocks[i].Ticker,
			preferredExchange: preferredExchange,
			stock:             &stocks[i],
		})
		seen[stocks[i].Ticker] = true
	}
	for _, b := range dashboardBenchmarks {
		if seen[b.ticker] {
			continue
		}
		targets = append(targets, priceSyncTarget{
			ticker:            b.ticker,
			preferredExchange: b.preferredExchange,
		})
	}
	return targets
}

func delay(ctx context.Context) {
	select {
	case <-time.After(syncCallDelay):
	case <-ctx.Done():
	}
}
