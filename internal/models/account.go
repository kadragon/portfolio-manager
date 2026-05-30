package models

import (
	"time"

	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

type Account struct {
	ID           uuidx.UUID
	Name         string
	CashBalance  numeric.Decimal
	CreatedAt    time.Time
	UpdatedAt    time.Time
	KisAccountNo *string
	KisAPIKeyID  *int64
}
