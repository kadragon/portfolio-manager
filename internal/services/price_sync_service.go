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

const (
	syncInterval  = 15 * time.Minute
	syncCallDelay = 200 * time.Millisecond
)

// PriceSyncService fetches prices for all stocks in the background and saves them to DB.
// It is the only component that calls PriceClient. PriceService reads from DB only.
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

// Start calls SyncOnce immediately, then repeats on a 15-minute interval
// until ctx is cancelled.
func (s *PriceSyncService) Start(ctx context.Context) {
	s.SyncOnce(ctx)
	t := time.NewTicker(syncInterval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			s.SyncOnce(ctx)
		case <-ctx.Done():
			return
		}
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
			delay(ctx, syncCallDelay)
		}
		quote, err := s.client.GetPrice(target.ticker, preferredExchange)
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
			delay(ctx, syncCallDelay)
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

func delay(ctx context.Context, d time.Duration) {
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}
