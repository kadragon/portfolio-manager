package models

import (
	"time"

	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// Asset class values. Drive account eligibility in the rebalance engine
// (IRP/연금 hold ETFs/funds only, never individual stocks).
const (
	AssetClassETF   = "etf"
	AssetClassStock = "stock"
)

// ValidAssetClass reports whether s is a recognized asset class.
func ValidAssetClass(s string) bool {
	switch s {
	case AssetClassETF, AssetClassStock:
		return true
	default:
		return false
	}
}

// KIS security-group codes (scty_grp_id_cd), normalized uppercase.
// Display/audit metadata only — not used by canHold eligibility logic.
const (
	SecurityGroupStock       = "ST" // 주식
	SecurityGroupDomesticETF = "EF" // 국내ETF
	SecurityGroupETN         = "EN" // ETN
	SecurityGroupELW         = "EW" // ELW
	SecurityGroupFund        = "MF" // 펀드
	SecurityGroupREIT        = "RT" // 리츠
	SecurityGroupForeignETF  = "FE" // 해외ETF
	SecurityGroupForeignStk  = "FS" // 해외주식
)

// ValidSecurityGroup reports whether s is a recognized KIS security-group code.
// Empty string is accepted (clears the field back to unclassified).
// KIS sync bypasses this check and writes codes directly.
func ValidSecurityGroup(s string) bool {
	switch s {
	case "",
		SecurityGroupStock, SecurityGroupDomesticETF, SecurityGroupETN,
		SecurityGroupELW, SecurityGroupFund, SecurityGroupREIT,
		SecurityGroupForeignETF, SecurityGroupForeignStk:
		return true
	default:
		return false
	}
}

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
