package models

import (
	"time"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/numeric"
)

// StockHolding pairs a stock with its aggregated quantity across all accounts.
type StockHolding struct {
	Stock    Stock
	Quantity numeric.Decimal
}

// StockHoldingWithPrice extends StockHolding with price and valuation data.
type StockHoldingWithPrice struct {
	Stock       Stock
	Quantity    numeric.Decimal
	Price       numeric.Decimal
	Currency    string
	Name        string
	ValueKRW    *numeric.Decimal           // nil when exchange rate unavailable
	ChangeRates map[string]numeric.Decimal // "1y","6m","1m","1d" → pct
}

// GroupHoldings is a group with its aggregated stock holdings.
type GroupHoldings struct {
	Group         Group
	StockHoldings []StockHolding
}

// GroupHoldingsWithPrice is a group with priced holdings.
type GroupHoldingsWithPrice struct {
	Group         Group
	StockHoldings []StockHoldingWithPrice
}

// PortfolioSummary aggregates valuation and return-rate data.
type PortfolioSummary struct {
	Holdings             []GroupHoldingPair
	TotalValue           numeric.Decimal
	TotalStockValue      numeric.Decimal
	TotalCashBalance     numeric.Decimal
	TotalAssets          numeric.Decimal
	TotalInvested        numeric.Decimal
	ReturnRate           *numeric.Decimal
	FirstDepositDate     *datex.Date
	AnnualizedReturnRate *numeric.Decimal
}

// GroupHoldingPair mirrors Python's list[tuple[Group, StockHoldingWithPrice]].
type GroupHoldingPair struct {
	Group   Group
	Holding StockHoldingWithPrice
}

// GroupSummaryRow is one row in the group allocation summary table.
type GroupSummaryRow struct {
	Group     Group
	Total     numeric.Decimal
	ActualPct numeric.Decimal
	TargetPct numeric.Decimal
	DiffPct   numeric.Decimal
	DiffVal   numeric.Decimal
}

// InvestmentAge returns elapsed days since first deposit. Returns -1 if unknown.
func (s *PortfolioSummary) InvestmentAge(today time.Time) int {
	if s.FirstDepositDate == nil {
		return -1
	}
	return int(today.Sub(s.FirstDepositDate.Time).Hours() / 24)
}
