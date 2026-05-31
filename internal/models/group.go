// Package models holds the domain types returned by repositories and consumed
// by services and the web layer, decoupled from the sqlc-generated row structs.
package models

import (
	"time"

	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// Group is a portfolio group with a target allocation percentage.
type Group struct {
	ID               uuidx.UUID
	Name             string
	TargetPercentage float64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
