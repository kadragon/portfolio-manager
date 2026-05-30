package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/repositories"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// ErrNoPriceService is returned by GetPortfolioSummary when no PriceService is wired.
var ErrNoPriceService = errors.New("price service not configured")

// PortfolioService aggregates portfolio data across groups, stocks, and holdings.
type PortfolioService struct {
	groups       *repositories.GroupRepository
	stocks       *repositories.StockRepository
	holdings     *repositories.HoldingRepository
	accounts     *repositories.AccountRepository
	deposits     *repositories.DepositRepository
	priceService *PriceService
	exchangeRate *ExchangeRateService
}

// NewPortfolioService builds a PortfolioService.
// priceService and exchangeRate may be nil (only GetHoldingsByGroup available).
func NewPortfolioService(
	groups *repositories.GroupRepository,
	stocks *repositories.StockRepository,
	holdings *repositories.HoldingRepository,
	accounts *repositories.AccountRepository,
	deposits *repositories.DepositRepository,
	priceService *PriceService,
	exchangeRate *ExchangeRateService,
) *PortfolioService {
	return &PortfolioService{
		groups:       groups,
		stocks:       stocks,
		holdings:     holdings,
		accounts:     accounts,
		deposits:     deposits,
		priceService: priceService,
		exchangeRate: exchangeRate,
	}
}

// HasPriceService reports whether a PriceService is configured.
func (s *PortfolioService) HasPriceService() bool {
	return s.priceService != nil
}

// GetHoldingsByGroup returns all groups with their aggregated holdings (qty > 0).
// Groups with no holdings are included with an empty StockHoldings list.
func (s *PortfolioService) GetHoldingsByGroup(ctx context.Context) ([]models.GroupHoldings, error) {
	groups, err := s.groups.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	allStocks, err := s.stocks.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	aggregated, err := s.holdings.GetAggregatedByStock(ctx)
	if err != nil {
		return nil, err
	}

	stocksByGroup := make(map[uuidx.UUID][]models.Stock)
	for _, stock := range allStocks {
		stocksByGroup[stock.GroupID] = append(stocksByGroup[stock.GroupID], stock)
	}

	result := make([]models.GroupHoldings, 0, len(groups))
	for _, group := range groups {
		stocks := stocksByGroup[group.ID]
		stockHoldings := make([]models.StockHolding, 0)
		for _, stock := range stocks {
			qty, ok := aggregated[stock.ID]
			if !ok || qty.IsZero() {
				continue
			}
			stockHoldings = append(stockHoldings, models.StockHolding{
				Stock:    stock,
				Quantity: qty,
			})
		}
		result = append(result, models.GroupHoldings{
			Group:         group,
			StockHoldings: stockHoldings,
		})
	}
	return result, nil
}

// GetPortfolioSummary computes a full portfolio summary using DB-cached prices.
// Returns ErrNoPriceService if no PriceService is configured.
func (s *PortfolioService) GetPortfolioSummary(ctx context.Context) (*models.PortfolioSummary, error) {
	if s.priceService == nil {
		return nil, ErrNoPriceService
	}

	groups, err := s.groups.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	allStocks, err := s.stocks.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	aggregated, err := s.holdings.GetAggregatedByStock(ctx)
	if err != nil {
		return nil, err
	}

	stocksByGroup := make(map[uuidx.UUID][]models.Stock)
	for _, stock := range allStocks {
		stocksByGroup[stock.GroupID] = append(stocksByGroup[stock.GroupID], stock)
	}

	var usdKRW *numeric.Decimal
	pairs := make([]models.GroupHoldingPair, 0)
	totalStockValue := numeric.Zero

	for _, group := range groups {
		for _, stock := range stocksByGroup[group.ID] {
			qty, ok := aggregated[stock.ID]
			if !ok || qty.IsZero() {
				continue
			}
			preferredExchange := ""
			if stock.Exchange != nil {
				preferredExchange = *stock.Exchange
			}
			price, currency, name, exchange := s.priceService.GetStockPrice(ctx, stock.Ticker, preferredExchange)
			if name == "" {
				name = stock.Name
			}
			holdingValue := numeric.Wrap(qty.Mul(price.Decimal))

			var valueKRW *numeric.Decimal
			if currency == "USD" {
				if s.exchangeRate == nil {
					v := numeric.Zero
					valueKRW = &v
				} else {
					if usdKRW == nil {
						r := s.exchangeRate.GetUSDKRW()
						usdKRW = &r
					}
					v := numeric.Wrap(holdingValue.Mul(usdKRW.Decimal))
					valueKRW = &v
				}
			} else {
				v := holdingValue
				valueKRW = &v
			}

			totalStockValue = numeric.Wrap(totalStockValue.Add(valueKRW.Decimal))

			_ = exchange // stored in stock after Phase 8 sync
			pairs = append(pairs, models.GroupHoldingPair{
				Group: group,
				Holding: models.StockHoldingWithPrice{
					Stock:    stock,
					Quantity: qty,
					Price:    price,
					Currency: currency,
					Name:     name,
					ValueKRW: valueKRW,
				},
			})
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		vi, vj := numeric.Zero, numeric.Zero
		if pairs[i].Holding.ValueKRW != nil {
			vi = *pairs[i].Holding.ValueKRW
		}
		if pairs[j].Holding.ValueKRW != nil {
			vj = *pairs[j].Holding.ValueKRW
		}
		return vi.GreaterThan(vj.Decimal)
	})

	totalCash := numeric.Zero
	if s.accounts != nil {
		accs, err := s.accounts.ListAll(ctx)
		if err == nil {
			for _, a := range accs {
				totalCash = numeric.Wrap(totalCash.Add(a.CashBalance.Decimal))
			}
		}
	}

	totalInvested := numeric.Zero
	var firstDate *datex.Date
	if s.deposits != nil {
		deps, err := s.deposits.ListAll(ctx)
		if err == nil {
			for _, d := range deps {
				totalInvested = numeric.Wrap(totalInvested.Add(d.Amount.Decimal))
			}
		}
		firstDate, _ = s.deposits.GetFirstDepositDate(ctx)
	}

	totalAssets := numeric.Wrap(totalStockValue.Add(totalCash.Decimal))

	var returnRate *numeric.Decimal
	var annualizedReturn *numeric.Decimal
	if totalInvested.IsPositive() {
		r := numeric.Wrap(totalAssets.Sub(totalInvested.Decimal).Div(totalInvested.Decimal).Mul(hundred.Decimal))
		returnRate = &r

		if firstDate != nil {
			daysElapsed := int(ktime.Now().Sub(firstDate.Time).Hours() / 24)
			if daysElapsed > 0 {
				ratio, _ := totalAssets.Div(totalInvested.Decimal).Float64()
				annualizedRatio := math.Pow(ratio, 365.0/float64(daysElapsed))
				ar, _ := numeric.FromString(fmt.Sprintf("%.10f", (annualizedRatio-1)*100))
				annualizedReturn = &ar
			}
		}
	}

	return &models.PortfolioSummary{
		Holdings:             pairs,
		TotalValue:           totalStockValue,
		TotalStockValue:      totalStockValue,
		TotalCashBalance:     totalCash,
		TotalAssets:          totalAssets,
		TotalInvested:        totalInvested,
		ReturnRate:           returnRate,
		FirstDepositDate:     firstDate,
		AnnualizedReturnRate: annualizedReturn,
	}, nil
}

// ComputeGroupSummary aggregates holdings by group for the allocation table.
func ComputeGroupSummary(summary *models.PortfolioSummary) []models.GroupSummaryRow {
	totals := make(map[uuidx.UUID]numeric.Decimal)
	groupByID := make(map[uuidx.UUID]models.Group)
	for _, pair := range summary.Holdings {
		if pair.Holding.ValueKRW == nil {
			continue
		}
		totals[pair.Group.ID] = numeric.Wrap(totals[pair.Group.ID].Add(pair.Holding.ValueKRW.Decimal))
		groupByID[pair.Group.ID] = pair.Group
	}

	denominator := summary.TotalStockValue
	rows := make([]models.GroupSummaryRow, 0, len(totals))
	for id, total := range totals {
		group := groupByID[id]
		var actualPct numeric.Decimal
		if denominator.IsPositive() {
			actualPct = numeric.Wrap(total.Div(denominator.Decimal).Mul(hundred.Decimal))
		}
		targetPct, _ := numeric.FromString(fmt.Sprintf("%g", group.TargetPercentage))
		diffPct := numeric.Wrap(actualPct.Sub(targetPct.Decimal))
		diffVal := numeric.Wrap(total.Sub(denominator.Mul(targetPct.Decimal).Div(hundred.Decimal)))
		rows = append(rows, models.GroupSummaryRow{
			Group:     group,
			Total:     total,
			ActualPct: actualPct,
			TargetPct: targetPct,
			DiffPct:   diffPct,
			DiffVal:   diffVal,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Total.GreaterThan(rows[j].Total.Decimal)
	})
	return rows
}

var hundred, _ = numeric.FromString("100")
