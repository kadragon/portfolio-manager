package repositories

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/kadragon/portfolio-manager/internal/db/sqlc"
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// HoldingUpdate pairs a holding ID with the new quantity for bulk updates.
type HoldingUpdate struct {
	ID       uuidx.UUID
	Quantity numeric.Decimal
}

// HoldingRepository provides holding CRUD operations.
type HoldingRepository struct {
	q *sqlc.Queries
}

// NewHoldingRepository builds a HoldingRepository over the given queries handle.
func NewHoldingRepository(q *sqlc.Queries) *HoldingRepository {
	return &HoldingRepository{q: q}
}

// Create inserts a new holding and returns it.
func (r *HoldingRepository) Create(ctx context.Context, accountID, stockID uuidx.UUID, quantity numeric.Decimal) (models.Holding, error) {
	now := ktime.Now()
	row, err := r.q.CreateHolding(ctx, sqlc.CreateHoldingParams{
		ID:        uuidx.New(),
		AccountID: accountID,
		StockID:   stockID,
		Quantity:  quantity,
		CreatedAt: now,
		UpdatedAt: now,
	})
	if err != nil {
		return models.Holding{}, err
	}
	return toHolding(row), nil
}

// ListByAccount returns all holdings for the given account in insertion order.
func (r *HoldingRepository) ListByAccount(ctx context.Context, accountID uuidx.UUID) ([]models.Holding, error) {
	rows, err := r.q.ListHoldingsByAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	out := make([]models.Holding, 0, len(rows))
	for _, row := range rows {
		out = append(out, toHolding(row))
	}
	return out, nil
}

// GetByID returns the holding with the given id, or nil if absent.
func (r *HoldingRepository) GetByID(ctx context.Context, id uuidx.UUID) (*models.Holding, error) {
	row, err := r.q.GetHoldingByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	h := toHolding(row)
	return &h, nil
}

// Update changes the quantity of a holding.
func (r *HoldingRepository) Update(ctx context.Context, id uuidx.UUID, quantity numeric.Decimal) (models.Holding, error) {
	row, err := r.q.UpdateHolding(ctx, sqlc.UpdateHoldingParams{
		Quantity:  quantity,
		UpdatedAt: ktime.Now(),
		ID:        id,
	})
	if err != nil {
		return models.Holding{}, err
	}
	return toHolding(row), nil
}

// Delete removes the holding with the given id.
func (r *HoldingRepository) Delete(ctx context.Context, id uuidx.UUID) error {
	return r.q.DeleteHolding(ctx, id)
}

// BulkUpdateByAccount validates that all updates belong to the account, then
// applies each update sequentially. Parity with Python: validation completes
// before any write; no transaction needed at this scale.
func (r *HoldingRepository) BulkUpdateByAccount(ctx context.Context, accountID uuidx.UUID, updates []HoldingUpdate) error {
	if len(updates) == 0 {
		return nil
	}

	existing, err := r.q.ListHoldingsByAccount(ctx, accountID)
	if err != nil {
		return err
	}
	existingIDs := make(map[uuidx.UUID]struct{}, len(existing))
	for _, h := range existing {
		existingIDs[h.ID] = struct{}{}
	}
	for _, u := range updates {
		if _, ok := existingIDs[u.ID]; !ok {
			return fmt.Errorf("all holdings must belong to account")
		}
	}

	now := ktime.Now()
	for _, u := range updates {
		if _, err := r.q.UpdateHolding(ctx, sqlc.UpdateHoldingParams{
			Quantity:  u.Quantity,
			UpdatedAt: now,
			ID:        u.ID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func toHolding(row sqlc.Holding) models.Holding {
	return models.Holding{
		ID:        row.ID,
		AccountID: row.AccountID,
		StockID:   row.StockID,
		Quantity:  row.Quantity,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}
}
