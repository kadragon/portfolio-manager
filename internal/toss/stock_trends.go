package toss

import (
	"context"
	"net/url"
	"strconv"
	"time"
)

// The five KR per-stock trend endpoints share one request shape: a KR
// symbol path segment plus optional count (max 100, API default 10) and
// until ("YYYY-MM-DD", inclusive upper bound; pass the previous response's
// NextUntil to page). Non-KR symbols are rejected by the API with 400
// unsupported-market. Every quantity/amount/rate is a decimal string.

// getStockTrend issues GET /api/v1/stocks/{symbol}/{kind}. count <= 0 and
// an empty until omit their params.
func getStockTrend[T any](ctx context.Context, c *Client, kind, symbol string, count int, until string) (T, error) {
	query := map[string]string{"until": until}
	if count > 0 {
		query["count"] = strconv.Itoa(count)
	}
	path := "/api/v1/stocks/" + url.PathEscape(symbol) + "/" + kind
	return doGet[T](ctx, c, "toss stock "+kind, path, query, nil)
}

// InvestorTradingVolume is a buy/sell/net-buy volume triple, in shares.
// NetBuyVolume is negative for net selling.
type InvestorTradingVolume struct {
	BuyVolume    string `json:"buyVolume"`
	SellVolume   string `json:"sellVolume"`
	NetBuyVolume string `json:"netBuyVolume"`
}

// StockInstitutionTradingBreakdown splits a stock's institutional volume
// into its 7 sub-categories.
type StockInstitutionTradingBreakdown struct {
	FinancialInvestment       InvestorTradingVolume `json:"financialInvestment"`
	Insurance                 InvestorTradingVolume `json:"insurance"`
	Trust                     InvestorTradingVolume `json:"trust"`
	PrivateEquityFund         InvestorTradingVolume `json:"privateEquityFund"`
	Bank                      InvestorTradingVolume `json:"bank"`
	OtherFinancialInstitution InvestorTradingVolume `json:"otherFinancialInstitution"`
	PensionFund               InvestorTradingVolume `json:"pensionFund"`
}

// StockInstitutionTradingVolume is the institutional total. Breakdown is nil
// on same-day provisional records.
type StockInstitutionTradingVolume struct {
	BuyVolume    string                            `json:"buyVolume"`
	SellVolume   string                            `json:"sellVolume"`
	NetBuyVolume string                            `json:"netBuyVolume"`
	Breakdown    *StockInstitutionTradingBreakdown `json:"breakdown"`
}

// ForeignerHolding is the foreign-ownership snapshot for a date.
// HoldingRate is a fraction of listed shares (0.5089 = 50.89%).
type ForeignerHolding struct {
	HoldingQuantity string `json:"holdingQuantity"`
	LimitQuantity   string `json:"limitQuantity"`
	HoldingRate     string `json:"holdingRate"`
}

// CfdBalance is the CFD balance for a date; rates are fractions of listed
// shares.
type CfdBalance struct {
	BuyBalanceQuantity  string `json:"buyBalanceQuantity"`
	BuyBalanceRate      string `json:"buyBalanceRate"`
	SellBalanceQuantity string `json:"sellBalanceQuantity"`
	SellBalanceRate     string `json:"sellBalanceRate"`
}

// StockInvestorTradingRecord is one day of per-investor volume. Individual,
// OtherCorporation and Institution.Breakdown are nil on same-day
// provisional records; ForeignerHolding and Cfd are nil until reflected.
type StockInvestorTradingRecord struct {
	Date             string                        `json:"date"`
	UpdatedAt        time.Time                     `json:"updatedAt"`
	Individual       *InvestorTradingVolume        `json:"individual"`
	Foreigner        InvestorTradingVolume         `json:"foreigner"`
	Institution      StockInstitutionTradingVolume `json:"institution"`
	OtherCorporation *InvestorTradingVolume        `json:"otherCorporation"`
	ForeignerHolding *ForeignerHolding             `json:"foreignerHolding"`
	Cfd              *CfdBalance                   `json:"cfd"`
}

// StockInvestorTradingResponse is the result of GetStockInvestorTrading.
type StockInvestorTradingResponse struct {
	NextUntil *string                      `json:"nextUntil"`
	Records   []StockInvestorTradingRecord `json:"records"`
}

// GetStockInvestorTrading returns daily per-investor volume (KRX+NXT) for a
// KR symbol, newest first.
func (c *Client) GetStockInvestorTrading(ctx context.Context, symbol string, count int, until string) (StockInvestorTradingResponse, error) {
	return getStockTrend[StockInvestorTradingResponse](ctx, c, "investor-trading", symbol, count, until)
}

// ProgramTradeRecord is one day of program-trading volume.
type ProgramTradeRecord struct {
	Date         string                `json:"date"`
	Arbitrage    InvestorTradingVolume `json:"arbitrage"`
	NonArbitrage InvestorTradingVolume `json:"nonArbitrage"`
}

// ProgramTradesResponse is the result of GetStockProgramTrades.
type ProgramTradesResponse struct {
	NextUntil *string              `json:"nextUntil"`
	Records   []ProgramTradeRecord `json:"records"`
}

// GetStockProgramTrades returns daily arbitrage/non-arbitrage program
// volume (KRX only) for a KR symbol, newest first.
func (c *Client) GetStockProgramTrades(ctx context.Context, symbol string, count int, until string) (ProgramTradesResponse, error) {
	return getStockTrend[ProgramTradesResponse](ctx, c, "program-trades", symbol, count, until)
}

// ShortSellingRecord is one day of short selling. The rates are fractions
// of the day's total volume/amount, nil when the denominator is missing.
type ShortSellingRecord struct {
	Date                   string    `json:"date"`
	UpdatedAt              time.Time `json:"updatedAt"`
	ShortSellingVolume     string    `json:"shortSellingVolume"`
	ShortSellingAmount     string    `json:"shortSellingAmount"`
	ShortSellingVolumeRate *string   `json:"shortSellingVolumeRate"`
	ShortSellingAmountRate *string   `json:"shortSellingAmountRate"`
}

// ShortSellingResponse is the result of GetStockShortSelling.
type ShortSellingResponse struct {
	NextUntil *string              `json:"nextUntil"`
	Records   []ShortSellingRecord `json:"records"`
}

// GetStockShortSelling returns daily short-selling volume/amount (KRW) for a
// KR symbol, newest first.
func (c *Client) GetStockShortSelling(ctx context.Context, symbol string, count int, until string) (ShortSellingResponse, error) {
	return getStockTrend[ShortSellingResponse](ctx, c, "short-selling", symbol, count, until)
}

// CreditTradeDetail is one credit-trade side for a date. BalanceRate is a
// fraction of listed shares; TradingRate a fraction of the day's volume.
type CreditTradeDetail struct {
	NewQuantity     string `json:"newQuantity"`
	ReturnQuantity  string `json:"returnQuantity"`
	BalanceQuantity string `json:"balanceQuantity"`
	BalanceRate     string `json:"balanceRate"`
	TradingRate     string `json:"tradingRate"`
}

// CreditTradeRecord is one day of credit trading. Either side is nil when
// that day has no data for it.
type CreditTradeRecord struct {
	Date       string             `json:"date"`
	UpdatedAt  time.Time          `json:"updatedAt"`
	MarginLoan *CreditTradeDetail `json:"marginLoan"`
	StockLoan  *CreditTradeDetail `json:"stockLoan"`
}

// CreditTradesResponse is the result of GetStockCreditTrades.
type CreditTradesResponse struct {
	NextUntil *string             `json:"nextUntil"`
	Records   []CreditTradeRecord `json:"records"`
}

// GetStockCreditTrades returns daily margin-loan/stock-loan activity for a
// KR symbol, newest first. Data lands T+1.
func (c *Client) GetStockCreditTrades(ctx context.Context, symbol string, count int, until string) (CreditTradesResponse, error) {
	return getStockTrend[CreditTradesResponse](ctx, c, "credit-trades", symbol, count, until)
}

// SecuritiesLendingRecord is one day of institutional securities lending.
// BalanceAmount is in KRW.
type SecuritiesLendingRecord struct {
	Date              string    `json:"date"`
	UpdatedAt         time.Time `json:"updatedAt"`
	ExecutionQuantity string    `json:"executionQuantity"`
	RepaymentQuantity string    `json:"repaymentQuantity"`
	BalanceQuantity   string    `json:"balanceQuantity"`
	BalanceAmount     string    `json:"balanceAmount"`
}

// SecuritiesLendingResponse is the result of GetStockSecuritiesLending.
type SecuritiesLendingResponse struct {
	NextUntil *string                   `json:"nextUntil"`
	Records   []SecuritiesLendingRecord `json:"records"`
}

// GetStockSecuritiesLending returns daily institutional lending activity for
// a KR symbol, newest first.
func (c *Client) GetStockSecuritiesLending(ctx context.Context, symbol string, count int, until string) (SecuritiesLendingResponse, error) {
	return getStockTrend[SecuritiesLendingResponse](ctx, c, "securities-lending", symbol, count, until)
}
