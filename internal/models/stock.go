package models

import (
	"time"

	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// Stock is a ticker held in a portfolio group.
type Stock struct {
	ID        uuidx.UUID
	Ticker    string
	GroupID   uuidx.UUID
	Exchange  *string // nil when unknown
	Name      string  // empty when not yet resolved
	CreatedAt time.Time
	UpdatedAt time.Time
}
