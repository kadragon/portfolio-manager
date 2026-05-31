package models

import (
	"time"

	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

type Holding struct {
	ID        uuidx.UUID
	AccountID uuidx.UUID
	StockID   uuidx.UUID
	Quantity  numeric.Decimal
	CreatedAt time.Time
	UpdatedAt time.Time
}
