package models

import (
	"database/sql"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// StockPrice is a cached price entry for a ticker on a given date.
type StockPrice struct {
	ID        uuidx.UUID
	Ticker    string
	Price     numeric.Decimal
	Currency  string
	Name      string
	Exchange  sql.NullString
	PriceDate datex.Date
	CreatedAt ktime.Time
	UpdatedAt ktime.Time
}
