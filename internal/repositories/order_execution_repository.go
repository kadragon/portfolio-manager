package repositories

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/kadragon/portfolio-manager/internal/db/sqlc"
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// OrderExecutionRepository persists order execution records.
type OrderExecutionRepository struct {
	q *sqlc.Queries
}

// NewOrderExecutionRepository builds the repository.
func NewOrderExecutionRepository(q *sqlc.Queries) *OrderExecutionRepository {
	return &OrderExecutionRepository{q: q}
}

// Create inserts a new order execution record.
func (r *OrderExecutionRepository) Create(
	ctx context.Context,
	ticker, side string,
	quantity int,
	currency, status, message string,
	exchange string,
	rawResponse map[string]any,
) (models.OrderExecutionRecord, error) {
	var exch sql.NullString
	if exchange != "" {
		exch = sql.NullString{String: exchange, Valid: true}
	}

	var rawJSON sql.NullString
	if rawResponse != nil {
		b, err := json.Marshal(rawResponse)
		if err != nil {
			return models.OrderExecutionRecord{}, err
		}
		rawJSON = sql.NullString{String: string(b), Valid: true}
	}

	row, err := r.q.CreateOrderExecution(ctx, sqlc.CreateOrderExecutionParams{
		ID:          uuidx.New(),
		Ticker:      ticker,
		Side:        side,
		Quantity:    int64(quantity),
		Currency:    currency,
		Exchange:    exch,
		Status:      status,
		Message:     message,
		RawResponse: rawJSON,
		CreatedAt:   ktime.Now(),
	})
	if err != nil {
		return models.OrderExecutionRecord{}, err
	}
	return toDomainOrderExecution(row), nil
}

// ListRecent returns the most recent executions ordered by created_at DESC.
func (r *OrderExecutionRepository) ListRecent(ctx context.Context, limit int) ([]models.OrderExecutionRecord, error) {
	rows, err := r.q.ListRecentOrderExecutions(ctx, int64(limit))
	if err != nil {
		return nil, err
	}
	result := make([]models.OrderExecutionRecord, 0, len(rows))
	for _, row := range rows {
		result = append(result, toDomainOrderExecution(row))
	}
	return result, nil
}

func toDomainOrderExecution(row sqlc.OrderExecution) models.OrderExecutionRecord {
	var raw map[string]any
	if row.RawResponse.Valid && row.RawResponse.String != "" {
		_ = json.Unmarshal([]byte(row.RawResponse.String), &raw)
	}

	exchange := ""
	if row.Exchange.Valid {
		exchange = row.Exchange.String
	}

	return models.OrderExecutionRecord{
		ID:          row.ID,
		Ticker:      row.Ticker,
		Side:        row.Side,
		Quantity:    int(row.Quantity),
		Currency:    row.Currency,
		Exchange:    exchange,
		Status:      row.Status,
		Message:     row.Message,
		RawResponse: raw,
		CreatedAt:   row.CreatedAt,
	}
}
