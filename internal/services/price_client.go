// Package services implements domain-level business logic.
package services

import "github.com/kadragon/portfolio-manager/internal/datex"

// PriceQuote is a price result from an external price source.
type PriceQuote struct {
	Symbol   string
	Name     string
	Price    float64
	Currency string
	Exchange string // may be empty
}

// PriceClient fetches live stock prices from an external source (e.g. KIS API).
// Implemented by KIS clients in Phase 6 continuation / Phase 8.
type PriceClient interface {
	// GetPrice returns the current quote for a ticker.
	// preferredExchange is the order-form exchange code (e.g. "NASD", "NYSE") or empty.
	GetPrice(ticker string, preferredExchange string) (PriceQuote, error)
	// GetHistoricalClose returns the closing price for a past date.
	GetHistoricalClose(ticker string, date datex.Date, preferredExchange string) (float64, error)
}
