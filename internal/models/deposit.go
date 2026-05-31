package models

import (
	"database/sql"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

type Deposit struct {
	ID          uuidx.UUID
	Amount      numeric.Decimal
	DepositDate datex.Date
	Note        sql.NullString
	CreatedAt   ktime.Time
	UpdatedAt   ktime.Time
}
