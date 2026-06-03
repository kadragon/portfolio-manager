package models

import (
	"time"

	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// Stock is a ticker held in a portfolio group.
type Stock struct {
	ID       uuidx.UUID
	Ticker   string
	GroupID  uuidx.UUID
	Exchange *string // nil when unknown
	Name     string  // empty when not yet resolved
	// AssetClass is "etf" or "stock". nil = unclassified. Drives account
	// eligibility (IRP/연금 hold only ETFs/funds, never individual stocks).
	AssetClass *string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
