package models

import (
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// KisHoldingPosition is a single holding returned by the KIS balance API.
type KisHoldingPosition struct {
	Ticker   string
	Quantity numeric.Decimal
	Name     string
}

// KisAccountSnapshot is the cash + holdings snapshot from the KIS balance API.
type KisAccountSnapshot struct {
	CashBalance         numeric.Decimal
	PreserveCashBalance bool
	Holdings            []KisHoldingPosition // sorted by ticker
}

// HoldingSyncDetail records one holding change during a sync operation.
type HoldingSyncDetail struct {
	Ticker      string
	Action      string // "created", "updated", "deleted"
	OldQuantity *numeric.Decimal
	NewQuantity *numeric.Decimal
}

// KisAccountSyncResult is the result summary of a KIS account sync.
type KisAccountSyncResult struct {
	AccountID         uuidx.UUID
	CashBalance       numeric.Decimal
	OldCashBalance    numeric.Decimal
	HoldingCount      int
	CreatedStockCount int
	HoldingChanges    []HoldingSyncDetail
}
