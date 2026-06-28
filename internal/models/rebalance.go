package models

import (
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// RebalanceAction is "buy" or "sell".
type RebalanceAction string

const (
	ActionBuy  RebalanceAction = "buy"
	ActionSell RebalanceAction = "sell"
)

// RebalanceRecommendation is one sell or buy directive produced by RebalanceService.
type RebalanceRecommendation struct {
	Ticker             string
	Action             RebalanceAction
	Amount             numeric.Decimal
	Priority           int
	Currency           string
	Quantity           *numeric.Decimal // nil when uncomputable
	StockName          string
	GroupName          string
	AccountID          uuidx.UUID
	AccountName        string
	RebalanceGroupName string
	Reason             string
	TriggerType        string
	AmountKRW          numeric.Decimal
	AmountLocal        numeric.Decimal
}

// GroupDiagnostic holds current/target/band status for one rebalance group.
type GroupDiagnostic struct {
	RebalanceGroupName string
	TargetPct          numeric.Decimal
	BandPct            numeric.Decimal
	LowerPct           numeric.Decimal
	UpperPct           numeric.Decimal
	CurrentPct         numeric.Decimal
	CurrentValueKRW    numeric.Decimal
	IsUpperBreached    bool
	IsLowerBreached    bool
}

// RegionDiagnostic holds KR/US region split status.
type RegionDiagnostic struct {
	TargetKRPct  numeric.Decimal
	TargetUSPct  numeric.Decimal
	CurrentKRPct numeric.Decimal
	CurrentUSPct numeric.Decimal
	LowerKRPct   numeric.Decimal
	UpperKRPct   numeric.Decimal
	IsTriggered  bool
}

// AccountRebalanceSummary collects per-account cash flow and recommendation lists.
type AccountRebalanceSummary struct {
	AccountID       uuidx.UUID
	AccountName     string
	StartingCashKRW numeric.Decimal
	SellCashKRW     numeric.Decimal
	TotalBuyKRW     numeric.Decimal
	UnusedCashKRW   numeric.Decimal
	UnmetGroups     []string
	SellRecs        []RebalanceRecommendation
	BuyRecs         []RebalanceRecommendation
}

// RebalancePlan is the complete output of RebalanceService.BuildPlan.
type RebalancePlan struct {
	SellRecs         []RebalanceRecommendation
	BuyRecs          []RebalanceRecommendation
	GroupDiagnostics []GroupDiagnostic
	RegionDiagnostic RegionDiagnostic
	TotalAssetsKRW   numeric.Decimal
	AccountSummaries []AccountRebalanceSummary
}

// OrderIntent is a standardized order request before sending to KIS.
type OrderIntent struct {
	Ticker      string
	Side        string // "buy" or "sell"
	Quantity    int
	Currency    string
	Exchange    string // overseas exchange code; "" for domestic
	StockName   string
	AccountID   uuidx.UUID
	AccountName string
	Amount      numeric.Decimal
}

// OrderExecutionResult is the result of executing one order.
type OrderExecutionResult struct {
	Intent      OrderIntent
	Status      string // "success", "failed", "skipped", "deferred"
	Message     string
	RawResponse map[string]any
}

// RebalanceExecutionResult is the full result of executing rebalance orders.
type RebalanceExecutionResult struct {
	Intents     []OrderIntent
	Skipped     []OrderIntent
	Deferred    []OrderIntent
	Executions  []OrderExecutionResult
	SyncWarning string
}

// OrderExecutionRecord is the persisted form of a KIS order execution.
type OrderExecutionRecord struct {
	ID          uuidx.UUID
	Ticker      string
	Side        string
	Quantity    int
	Currency    string
	Exchange    string
	Status      string
	Message     string
	RawResponse map[string]any
	CreatedAt   ktime.Time
}
