package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kadragon/portfolio-manager/internal/datex"
	"github.com/kadragon/portfolio-manager/internal/db/sqlc"
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// StockPriceRepository caches stock prices by ticker and date.
type StockPriceRepository struct {
	q *sqlc.Queries
}

// NewStockPriceRepository builds a StockPriceRepository.
func NewStockPriceRepository(q *sqlc.Queries) *StockPriceRepository {
	return &StockPriceRepository{q: q}
}

// GetByTickerAndDate returns the cached price for (ticker, date), or nil if absent.
func (r *StockPriceRepository) GetByTickerAndDate(ctx context.Context, ticker string, priceDate datex.Date) (*models.StockPrice, error) {
	row, err := r.q.GetStockPriceByTickerAndDate(ctx, sqlc.GetStockPriceByTickerAndDateParams{
		Ticker:    ticker,
		PriceDate: priceDate,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sp := toStockPrice(row)
	return &sp, nil
}

// GetLatestByTicker returns the most recently stored price for a ticker, or nil if absent.
func (r *StockPriceRepository) GetLatestByTicker(ctx context.Context, ticker string) (*models.StockPrice, error) {
	row, err := r.q.GetLatestStockPriceByTicker(ctx, ticker)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sp := toStockPrice(row)
	return &sp, nil
}

// Save upserts a price entry. If a row already exists for (ticker, date),
// it updates price/currency/exchange/updated_at and preserves the existing name
// when the new name is empty (matches Python behaviour).
func (r *StockPriceRepository) Save(
	ctx context.Context,
	ticker string,
	priceDate datex.Date,
	price numeric.Decimal,
	currency string,
	name string,
	exchange sql.NullString,
) (models.StockPrice, error) {
	now := ktime.Now()
	row, err := r.q.UpsertStockPrice(ctx, sqlc.UpsertStockPriceParams{
		ID:        uuidx.New(),
		Ticker:    ticker,
		Price:     price,
		Currency:  currency,
		Name:      name,
		Exchange:  exchange,
		PriceDate: priceDate,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return models.StockPrice{}, err
	}
	return toStockPrice(row), nil
}

func toStockPrice(row sqlc.StockPrice) models.StockPrice {
	return models.StockPrice{
		ID:        row.ID,
		Ticker:    row.Ticker,
		Price:     row.Price,
		Currency:  row.Currency,
		Name:      row.Name,
		Exchange:  row.Exchange,
		PriceDate: row.PriceDate,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
