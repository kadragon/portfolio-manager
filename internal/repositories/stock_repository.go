package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kadragon/portfolio-manager/internal/db/sqlc"
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// StockRepository provides stock CRUD operations.
type StockRepository struct {
	q *sqlc.Queries
}

// NewStockRepository builds a StockRepository over the given queries handle.
func NewStockRepository(q *sqlc.Queries) *StockRepository {
	return &StockRepository{q: q}
}

// Create inserts a new stock with an empty name and no exchange.
func (r *StockRepository) Create(ctx context.Context, ticker string, groupID uuidx.UUID) (models.Stock, error) {
	now := ktime.Now()
	row, err := r.q.CreateStock(ctx, sqlc.CreateStockParams{
		ID:        uuidx.New(),
		Ticker:    ticker,
		GroupID:   groupID,
		Exchange:  sql.NullString{Valid: false},
		CreatedAt: now,
		UpdatedAt: now,
		Name:      "",
	})
	if err != nil {
		return models.Stock{}, err
	}
	return toStock(row), nil
}

// ListByGroup returns all stocks for the given group in insertion order.
func (r *StockRepository) ListByGroup(ctx context.Context, groupID uuidx.UUID) ([]models.Stock, error) {
	rows, err := r.q.ListStocksByGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	out := make([]models.Stock, 0, len(rows))
	for _, row := range rows {
		out = append(out, toStock(row))
	}
	return out, nil
}

// ListAll returns every stock in insertion order.
func (r *StockRepository) ListAll(ctx context.Context) ([]models.Stock, error) {
	rows, err := r.q.ListAllStocks(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.Stock, 0, len(rows))
	for _, row := range rows {
		out = append(out, toStock(row))
	}
	return out, nil
}

// GetByID returns the stock with the given id, or nil if absent.
func (r *StockRepository) GetByID(ctx context.Context, id uuidx.UUID) (*models.Stock, error) {
	row, err := r.q.GetStockByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s := toStock(row)
	return &s, nil
}

// GetByTicker returns the stock with the given ticker, or nil if absent.
func (r *StockRepository) GetByTicker(ctx context.Context, ticker string) (*models.Stock, error) {
	row, err := r.q.GetStockByTicker(ctx, ticker)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s := toStock(row)
	return &s, nil
}

// UpdateTicker changes the stock's ticker symbol.
func (r *StockRepository) UpdateTicker(ctx context.Context, id uuidx.UUID, ticker string) (models.Stock, error) {
	row, err := r.q.UpdateStockTicker(ctx, sqlc.UpdateStockTickerParams{
		Ticker:    ticker,
		UpdatedAt: ktime.Now(),
		ID:        id,
	})
	if err != nil {
		return models.Stock{}, err
	}
	return toStock(row), nil
}

// UpdateGroup moves the stock to a different group.
func (r *StockRepository) UpdateGroup(ctx context.Context, id uuidx.UUID, groupID uuidx.UUID) (models.Stock, error) {
	row, err := r.q.UpdateStockGroup(ctx, sqlc.UpdateStockGroupParams{
		GroupID:   groupID,
		UpdatedAt: ktime.Now(),
		ID:        id,
	})
	if err != nil {
		return models.Stock{}, err
	}
	return toStock(row), nil
}

// UpdateExchange sets the stock's exchange field.
func (r *StockRepository) UpdateExchange(ctx context.Context, id uuidx.UUID, exchange string) (models.Stock, error) {
	row, err := r.q.UpdateStockExchange(ctx, sqlc.UpdateStockExchangeParams{
		Exchange:  sql.NullString{String: exchange, Valid: true},
		UpdatedAt: ktime.Now(),
		ID:        id,
	})
	if err != nil {
		return models.Stock{}, err
	}
	return toStock(row), nil
}

// UpdateName sets the stock's display name.
func (r *StockRepository) UpdateName(ctx context.Context, id uuidx.UUID, name string) (models.Stock, error) {
	row, err := r.q.UpdateStockName(ctx, sqlc.UpdateStockNameParams{
		Name:      name,
		UpdatedAt: ktime.Now(),
		ID:        id,
	})
	if err != nil {
		return models.Stock{}, err
	}
	return toStock(row), nil
}

// Delete removes the stock with the given id.
func (r *StockRepository) Delete(ctx context.Context, id uuidx.UUID) error {
	return r.q.DeleteStock(ctx, id)
}

func toStock(row sqlc.Stock) models.Stock {
	var exchange *string
	if row.Exchange.Valid {
		exchange = &row.Exchange.String
	}
	return models.Stock{
		ID:        row.ID,
		Ticker:    row.Ticker,
		GroupID:   row.GroupID,
		Exchange:  exchange,
		Name:      row.Name,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
