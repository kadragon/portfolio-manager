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
	// SecurityGroup is the KIS security-group classification (scty_grp_id_cd),
	// normalized uppercase: "ST"=주식, "EF"=국내ETF, "RT"=리츠, "EN"=ETN,
	// "EW"=ELW, "MF"=펀드, "FE"=해외ETF, "FS"=해외주식, etc. nil = unclassified.
	// Recorded for audit/display; finer-grained than AssetClass.
	SecurityGroup *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
