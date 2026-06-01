package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/ktime"
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
}

// NewPriceSyncService constructs the service. All deps are required.
func NewPriceSyncService(
	client PriceClient,
	stockPrices *repositories.StockPriceRepository,
	stocks *repositories.StockRepository,
) *PriceSyncService {
	return &PriceSyncService{
		client:      client,
		stockPrices: stockPrices,
		stocks:      stocks,
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

	today := datex.FromTime(ktime.NowKST())
	targetDates := computeTargetDates(today.Time)

	for idx, stock := range allStocks {
		if ctx.Err() != nil {
			return
		}

		preferredExchange := ""
		if stock.Exchange != nil {
			preferredExchange = *stock.Exchange
		}

		if idx > 0 {
			delay(ctx, syncCallDelay)
		}
		quote, err := s.client.GetPrice(stock.Ticker, preferredExchange)
		if err != nil {
			log.Printf("price sync: get price %s: %v", stock.Ticker, err)
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
		if _, err := s.stockPrices.Save(ctx, stock.Ticker, today, price, quote.Currency, quote.Name, exc); err != nil {
			log.Printf("price sync: save %s: %v", stock.Ticker, err)
		}

		// Persist resolved exchange (canonical form) to stock when it differs from the stored value.
		// canonical is the 4-letter code (NASD/NYSE/AMEX) expected by GetPrice/prioritizedExchanges.
		if quote.Exchange != "" {
			canonical := quote.Exchange
			if stock.Exchange == nil || *stock.Exchange != canonical {
				if _, err := s.stocks.UpdateExchange(ctx, stock.ID, canonical); err == nil {
					stock.Exchange = &canonical
					preferredExchange = canonical
				} else {
					log.Printf("price sync: update exchange %s: %v", stock.Ticker, err)
				}
			}
		}

		// Fetch any missing historical closes (fetch-once: immutable past data).
		for _, label := range []string{"1y", "6m", "1m", "1d"} {
			targetDate := datex.FromTime(targetDates[label])
			if cached, _ := s.stockPrices.GetByTickerAndDate(ctx, stock.Ticker, targetDate); cached != nil && cached.Price.IsPositive() {
				continue
			}
			delay(ctx, syncCallDelay)
			raw, histErr := s.client.GetHistoricalClose(stock.Ticker, targetDate, preferredExchange)
			if histErr != nil || raw <= 0 {
				continue
			}
			pastClose, parseErr := numeric.FromString(fmt.Sprintf("%g", raw))
			if parseErr != nil || !pastClose.IsPositive() {
				continue
			}
			if _, err := s.stockPrices.Save(ctx, stock.Ticker, targetDate, pastClose, quote.Currency, quote.Name, exc); err != nil {
				log.Printf("price sync: save historical %s: %v", stock.Ticker, err)
			}
		}
	}
}

func delay(ctx context.Context, d time.Duration) {
	select {
	case <-time.After(d):
	case <-ctx.Done():
	}
}
