package models

import (
	"time"

	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// Asset class values. Informational metadata only; no in-repo consumer
// currently uses asset_class to drive ETF or account eligibility.
const (
	AssetClassETF   = "etf"
	AssetClassStock = "stock"
	// AssetClassUnknown is the sentinel persisted onto a stock's asset_class when it
	// could not be resolved (classifier error or no signal). It is deliberately NOT
	// accepted by ValidAssetClass — it marks the absence of a class, not a class —
	// but it IS terminal for the classifier's "already classified" skip-gates, so a
	// sentinel-tagged ticker is not re-queried against KIS on every sync/ClassifyAll.
	// Stamped ONLY on asset_class, never on security_group, which keeps its KIS
	// scty_grp_id_cd value space. No in-repo consumer reads asset_class for ETF
	// gating today (the canHold gate was removed with rebalance_service.go in
	// PR #145), so any future one must treat a non-"etf" value — this sentinel
	// included — as non-ETF. Clearing asset_class to empty forces re-classification
	// on the next sync.
	AssetClassUnknown = "unknown"
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
// Does NOT accept empty string (mirrors ValidAssetClass). `pm stock update` accepts
// empty separately to clear the field; KIS sync bypasses this check entirely.
//
// KIS may add codes beyond the eight above, so callers that must not reject a
// legitimate future code should gate on WellFormedSecurityGroup and treat a
// false result here as "unknown", not "invalid".
func ValidSecurityGroup(s string) bool {
	switch s {
	case SecurityGroupStock, SecurityGroupDomesticETF, SecurityGroupETN,
		SecurityGroupELW, SecurityGroupFund, SecurityGroupREIT,
		SecurityGroupForeignETF, SecurityGroupForeignStk:
		return true
	default:
		return false
	}
}

// WellFormedSecurityGroup reports whether s has the shape of a KIS
// security-group code: exactly two uppercase ASCII letters. Every code KIS
// issues follows that shape, so this catches typos without rejecting codes
// added after this allowlist was written.
func WellFormedSecurityGroup(s string) bool {
	if len(s) != 2 {
		return false
	}
	for i := range len(s) {
		if s[i] < 'A' || s[i] > 'Z' {
			return false
		}
	}
	return true
}

// Stock is a ticker held in a portfolio group.
type Stock struct {
	ID       uuidx.UUID
	Ticker   string
	GroupID  uuidx.UUID
	Exchange *string // nil when unknown
	Name     string  // empty when not yet resolved
	// AssetClass is "etf", "stock", or the AssetClassUnknown sentinel.
	// nil = unclassified. Informational metadata feeding account-eligibility
	// reasoning (IRP/연금 hold only ETFs/funds, never individual stocks);
	// no in-repo gate consumes it today.
	AssetClass *string
	// SecurityGroup is the KIS security-group classification (scty_grp_id_cd),
	// normalized uppercase: "ST"=주식, "EF"=국내ETF, "RT"=리츠, "EN"=ETN,
	// "EW"=ELW, "MF"=펀드, "FE"=해외ETF, "FS"=해외주식, etc. nil = unclassified.
	// Recorded for audit/display; finer-grained than AssetClass.
	SecurityGroup *string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
