package models

import (
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// GroupDiagnostic holds current/target/band status for one rebalance group.
// Produced by the band-alert service; plan generation itself lives in the
// rebalance-plan agent skill, not in the app.
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
	OrderType   string
	Price       *numeric.Decimal
	RawResponse map[string]any
	CreatedAt   ktime.Time
}
