package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/repositories"
)

// syncCallDelay paces calls against KIS's rate limits. A var, not a const, so the
// tests can shorten it — the pacing is what makes a full sync slow, and paying it
// in unit tests buys nothing.
var syncCallDelay = 200 * time.Millisecond

// benchmarkDepositLookback bounds the walk-back when a deposit-date close comes
// back empty. prevBizDay only skips weekends, so a deposit made during a market
// closure still lands on a day with no close; without a fallback that deposit
// stays unpriced and the whole timing-matched benchmark reports nil. The bound is
// a window rather than an attempt count because Korean closures cluster: KRX was
// shut 2025-10-03 (개천절) through 2025-10-09 (한글날), five sessions, and the KRW
// benchmark proxies trade there. Two weeks clears the longest such run while
// staying inside maxBenchmarkPriceGap, so a close found inside it still
// legitimately stands in for the deposit.
const benchmarkDepositLookback = 14 * 24 * time.Hour

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

// SyncResult reports what one SyncOnce pass could not complete. A benchmark
// deposit date left unpriced is the one gap worth surfacing: timing-matched
// returns are all-or-nothing per benchmark, so a single unpriced deposit blanks
// the whole comparison, and price-sync would otherwise report plain success.
type SyncResult struct {
	UnpricedBenchmarkDates int
}

// SyncOnce fetches current prices for all stocks, then fills any missing
// historical closes: the 1y/6m/1m/1d checkpoints for every target, plus each
// deposit date for the dashboard benchmarks. Historical closes are never
// re-fetched once saved — past data is immutable.
func (s *PriceSyncService) SyncOnce(ctx context.Context) SyncResult {
	var result SyncResult
	allStocks, err := s.stocks.ListAll(ctx)
	if err != nil {
		log.Printf("price sync: list stocks: %v", err)
		return result
	}
	targets := syncTargets(allStocks)

	today := datex.FromTime(ktime.NowKST())
	baseDates := s.syncHistoricalDates(today)
	depositDates := s.benchmarkDepositDates(ctx, today)
	benchmarkTickers := benchmarkTickerSet()
	timingTickers := timingMatchedTickerSet()

	for idx, target := range targets {
		if ctx.Err() != nil {
			return result
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
		// Deposit dates are benchmark-only; held stocks sync base dates only.
		var targetDepositDates []datex.Date
		switch {
		case timingTickers[target.ticker]:
			// Timing-matched mode replays every deposit at its own date.
			targetDepositDates = depositDates
		case benchmarkTickers[target.ticker] && len(depositDates) > 0:
			// Dashboard-only proxies are compared from the first deposit alone, so
			// fetching the rest for them would be pure cost.
			targetDepositDates = depositDates[:1]
		}
		for _, targetDate := range baseDates {
			if _, canceled, _ := s.syncHistoricalClose(ctx, target.ticker, targetDate, preferredExchange, quote, exc); canceled {
				return result
			}
		}
		for _, depositDate := range targetDepositDates {
			priced, canceled := s.ensureBenchmarkDepositClose(ctx, target.ticker, depositDate, preferredExchange, quote, exc)
			if canceled {
				return result
			}
			if !priced {
				result.UnpricedBenchmarkDates++
			}
		}
	}

	return result
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

// syncHistoricalDates returns the fixed 1y/6m/1m/1d checkpoints synced for every
// target (held stocks and benchmarks alike).
func (s *PriceSyncService) syncHistoricalDates(today datex.Date) []datex.Date {
	targetDates := computeTargetDates(today.Time)
	dates := make([]datex.Date, 0, len(targetDates))
	for _, label := range []string{"1y", "6m", "1m", "1d"} {
		dates = append(dates, datex.FromTime(targetDates[label]))
	}
	return dates
}

// benchmarkDepositDates returns every deposit date (prev-business-day adjusted),
// ascending — checkpoints only the dashboard benchmarks need. The first backs
// GetStockChangeSince's benchmark-vs-portfolio comparison; the rest back the
// timing-matched mode, which prices each deposit at its own date and reports nil
// unless every one of them is priceable, so the sparse 1y/6m/1m/1d checkpoints
// leave multi-year holes that maxBenchmarkPriceGap rejects. Held stocks need none
// of these, so they stay out of the base set.
//
// Dates are deduped against each other but deliberately NOT against the base
// checkpoints: a deposit that lands on a base date still needs
// ensureBenchmarkDepositClose's walk-back if that date turns out to be a market
// holiday, and the cache checks downstream make the overlap free.
func (s *PriceSyncService) benchmarkDepositDates(ctx context.Context, today datex.Date) []datex.Date {
	if s.deposits == nil {
		return nil
	}
	deposits, err := s.deposits.ListAll(ctx)
	if err != nil {
		log.Printf("price sync: list deposits: %v", err)
		return nil
	}

	seen := make(map[string]bool, len(deposits))
	dates := make([]datex.Date, 0, len(deposits))
	for _, d := range deposits {
		if d.DepositDate.Time.IsZero() || d.DepositDate.ISO() >= today.ISO() {
			continue
		}
		adjusted := datex.FromTime(prevBizDay(d.DepositDate.Time))
		if seen[adjusted.ISO()] {
			continue
		}
		seen[adjusted.ISO()] = true
		dates = append(dates, adjusted)
	}
	if len(dates) == 0 {
		return nil
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].ISO() < dates[j].ISO() })
	return dates
}

// ensureBenchmarkDepositClose makes depositDate priceable for ticker. A cached
// close already sitting inside the lookback window is what timingMatchedReturn
// actually reads, so it returns without a call — which is also what keeps a
// second SyncOnce from re-requesting a market-holiday date whose close is stored
// a few days behind it. Otherwise it fetches, walking back over a closure.
// priced reports whether a close now stands for the date; canceled reports that
// ctx ended. A close that cannot be found is logged and left absent, since an
// unpriced date is preferable to a price stored under a date the market never
// traded on.
func (s *PriceSyncService) ensureBenchmarkDepositClose(
	ctx context.Context,
	ticker string,
	depositDate datex.Date,
	preferredExchange string,
	quote PriceQuote,
	exc sql.NullString,
) (priced, canceled bool) {
	floor := depositDate.Time.Add(-benchmarkDepositLookback)
	if cached, err := s.stockPrices.GetOnOrBeforeDate(ctx, ticker, depositDate); err == nil &&
		cached != nil && cached.Price.IsPositive() && !cached.PriceDate.Time.Before(floor) {
		return true, false
	}

	for candidate := depositDate; !candidate.Time.Before(floor); candidate = datex.FromTime(prevBizDay(candidate.Time.AddDate(0, 0, -1))) {
		ok, canceled, fetchFailed := s.syncHistoricalClose(ctx, ticker, candidate, preferredExchange, quote, exc)
		if canceled {
			return false, true
		}
		if ok {
			return true, false
		}
		if fetchFailed {
			// A transport or storage failure is not evidence the market was shut.
			// Walking back on it would multiply calls against an API already
			// failing, so stop and let the next sync retry from the top.
			log.Printf("price sync: benchmark deposit close %s %s: fetch failed, left unpriced", ticker, depositDate.ISO())
			return false, false
		}
	}

	log.Printf("price sync: benchmark deposit close %s %s: no close within %d days, left unpriced",
		ticker, depositDate.ISO(), int(benchmarkDepositLookback.Hours()/24))
	return false, false
}

// syncHistoricalClose ensures one (ticker, date) close is on record: a date
// already cached is left alone (past data is immutable), otherwise the close is
// fetched and saved. ok reports whether a close now stands for that exact date.
// canceled reports that ctx ended mid-call and the whole pass should stop.
// fetchFailed separates "the call or the save failed" from "the market returned
// no close that day", which is the only outcome a walk-back should act on.
func (s *PriceSyncService) syncHistoricalClose(
	ctx context.Context,
	ticker string,
	targetDate datex.Date,
	preferredExchange string,
	quote PriceQuote,
	exc sql.NullString,
) (ok, canceled, fetchFailed bool) {
	if cached, _ := s.stockPrices.GetByTickerAndDate(ctx, ticker, targetDate); cached != nil && cached.Price.IsPositive() {
		return true, false, false
	}
	delay(ctx)
	if ctx.Err() != nil {
		return false, true, false
	}
	raw, histErr := s.client.GetHistoricalClose(ticker, targetDate, preferredExchange)
	if histErr != nil {
		return false, false, true
	}
	if raw <= 0 {
		// The market was shut that day (or KIS has no row for it) — the one case
		// worth walking back from.
		return false, false, false
	}
	pastClose, parseErr := numeric.FromString(fmt.Sprintf("%g", raw))
	if parseErr != nil || !pastClose.IsPositive() {
		return false, false, true
	}
	if _, err := s.stockPrices.Save(ctx, ticker, targetDate, pastClose, quote.Currency, quote.Name, exc); err != nil {
		log.Printf("price sync: save historical %s: %v", ticker, err)
		return false, false, true
	}
	return true, false, false
}

// benchmarkTickerSet is the set of dashboard-benchmark tickers, used to decide
// which sync targets also need the benchmark-only historical dates.
func benchmarkTickerSet() map[string]bool {
	specs := allBenchmarks()
	set := make(map[string]bool, len(specs))
	for _, b := range specs {
		set[b.ticker] = true
	}
	return set
}

// timingMatchedTickerSet is the subset of benchmark tickers the timing-matched
// mode replays deposits into. Only these need a checkpoint at every deposit date;
// the dashboard-only proxies are measured from the first deposit alone.
func timingMatchedTickerSet() map[string]bool {
	set := make(map[string]bool, len(timingMatchedBenchmarks))
	for _, b := range timingMatchedBenchmarks {
		set[b.ticker] = true
	}
	return set
}

type priceSyncTarget struct {
	ticker            string
	preferredExchange string
	stock             *models.Stock
}

func syncTargets(stocks []models.Stock) []priceSyncTarget {
	benchmarks := allBenchmarks()
	targets := make([]priceSyncTarget, 0, len(stocks)+len(benchmarks))
	seen := make(map[string]bool, len(stocks)+len(benchmarks))
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
	for _, b := range benchmarks {
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
