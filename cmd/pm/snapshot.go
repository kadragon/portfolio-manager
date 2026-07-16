package main

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"time"

	"github.com/kadragon/portfolio-manager/internal/container"
	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
	"github.com/shopspring/decimal"
)

// snapshotHolding is one priced position within an account.
type snapshotHolding struct {
	Ticker   string  `json:"ticker"`
	Name     string  `json:"name"`
	Exchange *string `json:"exchange"`
	Group    string  `json:"group"`
	Qty      float64 `json:"qty"`
	Price    float64 `json:"price"`
	Currency *string `json:"currency"`
	ValueKRW int64   `json:"value_krw"`
}

// snapshotAccount is one account with its cash and holdings.
type snapshotAccount struct {
	Name     string            `json:"name"`
	Type     *string           `json:"type"`
	CashKRW  float64           `json:"cash_krw"`
	Holdings []snapshotHolding `json:"holdings"`
}

// snapshotGroup is one group's aggregated value and weight.
type snapshotGroup struct {
	Name        string   `json:"name"`
	DBTargetPct *float64 `json:"db_target_pct"`
	ValueKRW    int64    `json:"value_krw"`
	WeightPct   *float64 `json:"weight_pct"`
}

// snapshotOutput mirrors the JSON shape the rebalance-plan skill scripts
// (check_deviations.py, verify_plan.py) consume. Field order matches the
// former snapshot.py output so downstream diffs stay quiet.
type snapshotOutput struct {
	AsOf             string            `json:"as_of"`
	FxUSDKRW         float64           `json:"fx_usdkrw"`
	TotalKRW         int64             `json:"total_krw"`
	TotalHoldingsKRW int64             `json:"total_holdings_krw"`
	TotalCashKRW     int64             `json:"total_cash_krw"`
	Accounts         []snapshotAccount `json:"accounts"`
	Groups           []snapshotGroup   `json:"groups"`
	Warnings         []string          `json:"warnings"`
}

// runSnapshot prints a portfolio snapshot for rebalance planning: holdings
// valued at their latest cached price (USD converted at -fx), aggregated by
// group, plus per-account cash. Deterministic groundwork for the rebalance-plan
// skill — trade planning and tax judgment stay with the agent. Replaces the old
// snapshot.py, which read the SQLite DB directly and so bypassed the repository
// layer (Golden Principle #1).
func runSnapshot(ctx context.Context, c *container.Container, args []string) error {
	fs := flag.NewFlagSet("pm snapshot", flag.ExitOnError)
	fx := fs.Float64("fx", 0, "USD/KRW rate (required)")
	staleDays := fs.Int("stale-days", 7, "warn when the latest price is older than this many days")
	if err := parseFlags(fs, args); err != nil {
		return err
	}
	if *fx <= 0 {
		return fmt.Errorf("-fx is required and must be positive")
	}

	out, err := buildSnapshot(ctx, c, *fx, *staleDays)
	if err != nil {
		return err
	}
	return printJSON(out)
}

// buildSnapshot assembles the snapshot from the repository layer. Split from
// runSnapshot (flags + printing) so the aggregation is unit-testable.
func buildSnapshot(ctx context.Context, c *container.Container, fx float64, staleDays int) (snapshotOutput, error) {
	fxDec := decimal.NewFromFloat(fx)

	accounts, err := c.Accounts.ListAll(ctx)
	if err != nil {
		return snapshotOutput{}, fmt.Errorf("list accounts: %w", err)
	}
	groups, err := c.Groups.ListAll(ctx)
	if err != nil {
		return snapshotOutput{}, fmt.Errorf("list groups: %w", err)
	}
	stocks, err := c.Stocks.ListAll(ctx)
	if err != nil {
		return snapshotOutput{}, fmt.Errorf("list stocks: %w", err)
	}
	holdings, err := c.Holdings.ListAll(ctx)
	if err != nil {
		return snapshotOutput{}, fmt.Errorf("list holdings: %w", err)
	}

	// uuidx.UUID is a comparable fixed-size array, so it keys these maps directly
	// — no per-lookup string allocation.
	stockByID := make(map[uuidx.UUID]models.Stock, len(stocks))
	for _, s := range stocks {
		stockByID[s.ID] = s
	}
	groupByID := make(map[uuidx.UUID]models.Group, len(groups))
	for _, g := range groups {
		groupByID[g.ID] = g
	}

	// acctIndex maps an account ID to its slot in out.Accounts, populated in
	// accounts (repository) order; holdings are attached to their slot below.
	acctIndex := make(map[uuidx.UUID]int, len(accounts))
	out := snapshotOutput{
		AsOf:     datex.FromTime(ktime.NowKST()).ISO(),
		FxUSDKRW: fx,
		Accounts: []snapshotAccount{},
		Groups:   []snapshotGroup{},
		Warnings: []string{},
	}
	for _, a := range accounts {
		acctIndex[a.ID] = len(out.Accounts)
		out.Accounts = append(out.Accounts, snapshotAccount{
			Name:     a.Name,
			Type:     a.AccountType,
			CashKRW:  a.CashBalance.InexactFloat64(),
			Holdings: []snapshotHolding{},
		})
	}

	// Seed every group at 0 so a target-only group (no holdings yet, e.g. a
	// freshly added 금/채권 allocation) still surfaces its underweight.
	groupTotals := make(map[string]decimal.Decimal, len(groups))
	for _, g := range groups {
		groupTotals[g.Name] = decimal.Zero
	}

	priceCache := make(map[string]*models.StockPrice)
	priced := make(map[string]bool)
	today := datex.FromTime(ktime.NowKST())

	for _, h := range holdings {
		// Skip holdings whose stock/group/account rows are absent, mirroring the
		// former snapshot.py's INNER JOINs (FK constraints make this unreachable
		// in a consistent DB; the guards keep a transient inconsistency from
		// panicking or silently misattributing a holding to the wrong account).
		stock, ok := stockByID[h.StockID]
		if !ok {
			continue
		}
		grp, ok := groupByID[stock.GroupID]
		if !ok {
			continue
		}
		acctSlot, ok := acctIndex[h.AccountID]
		if !ok {
			continue
		}

		var priceF float64
		var currency *string
		value := decimal.Zero

		if !priced[stock.Ticker] {
			p, perr := c.StockPrices.GetLatestByTicker(ctx, stock.Ticker)
			if perr != nil {
				return snapshotOutput{}, fmt.Errorf("latest price %s: %w", stock.Ticker, perr)
			}
			priceCache[stock.Ticker] = p
			priced[stock.Ticker] = true
		}
		p := priceCache[stock.Ticker]

		switch {
		case p == nil:
			out.Warnings = append(out.Warnings,
				fmt.Sprintf("no price for %s — value set to 0", stock.Ticker))
		case !p.Price.IsPositive():
			cur := p.Currency
			currency = &cur
			out.Warnings = append(out.Warnings, fmt.Sprintf(
				"non-positive price %s for %s (%s) — likely bad data, value set to 0, "+
					"do not trust this group's weight",
				p.Price.String(), stock.Ticker, p.PriceDate.ISO()))
		default:
			cur := p.Currency
			currency = &cur
			priceF = p.Price.InexactFloat64()
			if age := daysBetween(p.PriceDate, today); age > staleDays {
				out.Warnings = append(out.Warnings, fmt.Sprintf(
					"price for %s is %d days old (%s)", stock.Ticker, age, p.PriceDate.ISO()))
			}
			value = h.Quantity.Mul(p.Price.Decimal)
			if p.Currency == "USD" {
				value = value.Mul(fxDec)
			}
		}

		valueKRW := value.Round(0).IntPart()
		out.Accounts[acctSlot].Holdings = append(
			out.Accounts[acctSlot].Holdings, snapshotHolding{
				Ticker:   stock.Ticker,
				Name:     stock.Name,
				Exchange: stock.Exchange,
				Group:    grp.Name,
				Qty:      h.Quantity.InexactFloat64(),
				Price:    priceF,
				Currency: currency,
				ValueKRW: valueKRW,
			})
		groupTotals[grp.Name] = groupTotals[grp.Name].Add(value)
	}

	totalHoldings := decimal.Zero
	for _, v := range groupTotals {
		totalHoldings = totalHoldings.Add(v)
	}
	totalCash := decimal.Zero
	for _, a := range accounts {
		totalCash = totalCash.Add(a.CashBalance.Decimal)
	}
	total := totalHoldings.Add(totalCash)

	out.TotalKRW = total.Round(0).IntPart()
	out.TotalHoldingsKRW = totalHoldings.Round(0).IntPart()
	out.TotalCashKRW = totalCash.Round(0).IntPart()

	// Accounts: largest holdings value first, name-tiebroken so equal-value
	// accounts keep a stable order run-to-run (accounts ListAll has no ORDER BY).
	sort.SliceStable(out.Accounts, func(i, j int) bool {
		vi, vj := accountHoldingsValue(out.Accounts[i]), accountHoldingsValue(out.Accounts[j])
		if vi != vj {
			return vi > vj
		}
		return out.Accounts[i].Name < out.Accounts[j].Name
	})

	// Groups: largest value first, carrying each group's DB target and weight.
	// Iterate the groups slice (not the groupTotals map, whose iteration order
	// Go randomizes) so equal-valued groups — e.g. target-only groups all at 0 —
	// keep a deterministic order; name-tiebreak the value sort for the same reason.
	type gv struct {
		name string
		val  decimal.Decimal
	}
	ordered := make([]gv, 0, len(groups))
	for _, g := range groups {
		ordered = append(ordered, gv{g.Name, groupTotals[g.Name]})
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		if !ordered[i].val.Equal(ordered[j].val) {
			return ordered[i].val.GreaterThan(ordered[j].val)
		}
		return ordered[i].name < ordered[j].name
	})
	hundred := decimal.NewFromInt(100)
	for _, g := range ordered {
		var target *float64
		if grp, ok := groupNameTarget(groups, g.name); ok {
			target = &grp
		}
		var weight *float64
		if total.IsPositive() {
			w, _ := hundred.Mul(g.val).Div(total).Round(2).Float64()
			weight = &w
		}
		out.Groups = append(out.Groups, snapshotGroup{
			Name:        g.name,
			DBTargetPct: target,
			ValueKRW:    g.val.Round(0).IntPart(),
			WeightPct:   weight,
		})
	}

	return out, nil
}

// daysBetween returns the whole-day gap between two calendar dates. It reads
// each date's Y/M/D and re-pins both to UTC midnight before subtracting, so the
// result is independent of the timezone the driver happened to attach to a
// scanned price_date (which need not match today's KST midnight).
func daysBetween(earlier, later datex.Date) int {
	e := time.Date(earlier.Year(), earlier.Month(), earlier.Day(), 0, 0, 0, 0, time.UTC)
	l := time.Date(later.Year(), later.Month(), later.Day(), 0, 0, 0, 0, time.UTC)
	return int(l.Sub(e).Hours() / 24)
}

// accountHoldingsValue sums a snapshot account's holdings value for sorting.
func accountHoldingsValue(a snapshotAccount) int64 {
	var sum int64
	for _, h := range a.Holdings {
		sum += h.ValueKRW
	}
	return sum
}

// groupNameTarget looks up a group's target_percentage by name.
func groupNameTarget(groups []models.Group, name string) (float64, bool) {
	for _, g := range groups {
		if g.Name == name {
			return g.TargetPercentage, true
		}
	}
	return 0, false
}
