// Package container wires the repositories (and, in later phases, services and
// external clients) over an open database, the Go counterpart of
// core/container.py's ServiceContainer. As the composition root it may depend on
// both the db and repository layers.
package container

import (
	"database/sql"

	"github.com/kadragon/portfolio-manager/internal/db"
	"github.com/kadragon/portfolio-manager/internal/db/sqlc"
	"github.com/kadragon/portfolio-manager/internal/repositories"
	"github.com/kadragon/portfolio-manager/internal/services"
)

// Container holds shared dependencies for the web layer.
type Container struct {
	DB          *sql.DB
	Groups      *repositories.GroupRepository
	Stocks      *repositories.StockRepository
	Accounts    *repositories.AccountRepository
	Holdings    *repositories.HoldingRepository
	Deposits    *repositories.DepositRepository
	StockPrices *repositories.StockPriceRepository
	Portfolio   *services.PortfolioService
	// PriceService and ExchangeRate are nil until KIS API keys are configured
	// (Phase 6 continuation). Dashboard falls back to group_holdings when nil.
}

// New opens the database at path (empty = default location) and builds the
// repositories. The caller is responsible for Close.
func New(path string) (*Container, error) {
	sqlDB, q, err := db.Open(path)
	if err != nil {
		return nil, err
	}
	return NewWithQueries(sqlDB, q), nil
}

// NewWithQueries builds a Container over an already-open database and queries
// handle (used by tests with an in-memory database).
func NewWithQueries(sqlDB *sql.DB, q *sqlc.Queries) *Container {
	groups := repositories.NewGroupRepository(q)
	stocks := repositories.NewStockRepository(q)
	accounts := repositories.NewAccountRepository(q)
	holdings := repositories.NewHoldingRepository(q)
	deposits := repositories.NewDepositRepository(q)
	stockPrices := repositories.NewStockPriceRepository(q)

	// PriceService is DB-only (no KIS client); shows dashboard summary using cached prices.
	// KIS HTTP client wired after KIS stack is implemented (Phase 6 continuation).
	priceService := services.NewPriceService(stockPrices, nil)
	portfolio := services.NewPortfolioService(groups, stocks, holdings, accounts, deposits, priceService, nil)

	return &Container{
		DB:          sqlDB,
		Groups:      groups,
		Stocks:      stocks,
		Accounts:    accounts,
		Holdings:    holdings,
		Deposits:    deposits,
		StockPrices: stockPrices,
		Portfolio:   portfolio,
	}
}

// Close releases the database connection.
func (c *Container) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}
