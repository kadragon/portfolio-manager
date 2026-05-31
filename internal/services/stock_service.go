package services

import (
	"context"
	"log"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/repositories"
	"github.com/kadragon/portfolio-manager/internal/stockformat"
)

// StockService resolves and persists stock display names.
// Mirrors Python's StockService.
type StockService struct {
	stocks       *repositories.StockRepository
	priceService *PriceService
}

// NewStockService creates a StockService. priceService may be nil.
func NewStockService(stocks *repositories.StockRepository, priceService *PriceService) *StockService {
	return &StockService{stocks: stocks, priceService: priceService}
}

// SetPriceService wires in a PriceService after construction (late-binding).
func (s *StockService) SetPriceService(ps *PriceService) {
	s.priceService = ps
}

// PersistName formats raw_name and persists it to the DB when stock.Name is empty.
// Returns the formatted raw_name if non-empty, else the formatted stored name.
func (s *StockService) PersistName(ctx context.Context, stock *models.Stock, rawName string) string {
	formatted := stockformat.FormatName(rawName)
	if stock.Name == "" && formatted != "" {
		updated, err := s.stocks.UpdateName(ctx, stock.ID, formatted)
		if err != nil {
			log.Printf("StockService: persist name for %s: %v", stock.Ticker, err)
		} else {
			stock.Name = updated.Name
			stock.UpdatedAt = updated.UpdatedAt
		}
	}
	if formatted != "" {
		return formatted
	}
	return stockformat.FormatName(stock.Name)
}

// ResolveAndPersistName returns the formatted display name, persisting it when newly resolved.
// Returns the formatted existing name if already set.
// Calls priceService when stock.Name is empty; returns "" if unavailable.
func (s *StockService) ResolveAndPersistName(ctx context.Context, stock *models.Stock) string {
	if stock.Name != "" {
		return stockformat.FormatName(stock.Name)
	}
	if s.priceService == nil {
		return ""
	}

	preferredExchange := ""
	if stock.Exchange != nil {
		preferredExchange = *stock.Exchange
	}
	_, _, resolvedName, _ := s.priceService.GetStockPrice(ctx, stock.Ticker, preferredExchange)
	return s.PersistName(ctx, stock, resolvedName)
}
