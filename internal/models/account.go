package models

import (
	"time"

	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// Account type values. Drive tax/eligibility rules in the rebalance engine.
const (
	AccountTypeBrokerage = "brokerage" // 위탁: anything
	AccountTypeIRP       = "irp"       // IRP: domestic-listed ETFs/funds only
	AccountTypePension   = "pension"   // 연금저축: domestic-listed ETFs/funds only
	AccountTypeISA       = "isa"       // ISA(중개형): domestic-listed (ETF or stock)
)

// ValidAccountType reports whether s is a recognized account type.
func ValidAccountType(s string) bool {
	switch s {
	case AccountTypeBrokerage, AccountTypeIRP, AccountTypePension, AccountTypeISA:
		return true
	default:
		return false
	}
}

type Account struct {
	ID           uuidx.UUID
	Name         string
	CashBalance  numeric.Decimal
	CreatedAt    time.Time
	UpdatedAt    time.Time
	KisAccountNo *string
	KisAPIKeyID  *int64
	// AccountType is the tax/eligibility class: "brokerage", "irp", "pension"
	// (연금저축), or "isa". nil = unclassified (treated strictly: buys blocked).
	AccountType *string
}
