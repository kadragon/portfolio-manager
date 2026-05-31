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

// DepositRepository provides deposit CRUD operations.
type DepositRepository struct {
	q *sqlc.Queries
}

// NewDepositRepository builds a DepositRepository over the given queries handle.
func NewDepositRepository(q *sqlc.Queries) *DepositRepository {
	return &DepositRepository{q: q}
}

// Create inserts a new deposit.
func (r *DepositRepository) Create(ctx context.Context, amount numeric.Decimal, depositDate datex.Date, note sql.NullString) (models.Deposit, error) {
	now := ktime.Now()
	row, err := r.q.CreateDeposit(ctx, sqlc.CreateDepositParams{
		ID:          uuidx.New(),
		Amount:      amount,
		DepositDate: depositDate,
		Note:        note,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return models.Deposit{}, err
	}
	return toDeposit(row), nil
}

// ListAll returns all deposits ordered by deposit_date DESC.
func (r *DepositRepository) ListAll(ctx context.Context) ([]models.Deposit, error) {
	rows, err := r.q.ListDeposits(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.Deposit, 0, len(rows))
	for _, row := range rows {
		out = append(out, toDeposit(row))
	}
	return out, nil
}

// GetByID returns a deposit by ID, or nil if not found.
func (r *DepositRepository) GetByID(ctx context.Context, id uuidx.UUID) (*models.Deposit, error) {
	row, err := r.q.GetDepositByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	d := toDeposit(row)
	return &d, nil
}

// GetByDate returns a deposit matching the given date, or nil if not found.
func (r *DepositRepository) GetByDate(ctx context.Context, date datex.Date) (*models.Deposit, error) {
	row, err := r.q.GetDepositByDate(ctx, date)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	d := toDeposit(row)
	return &d, nil
}

// Update updates amount, deposit_date, and optionally note.
// note==nil keeps the existing note unchanged; non-nil sets it (NullString).
func (r *DepositRepository) Update(ctx context.Context, id uuidx.UUID, amount numeric.Decimal, depositDate datex.Date, note *sql.NullString) (models.Deposit, error) {
	now := ktime.Now()
	var row sqlc.Deposit
	var err error
	if note == nil {
		row, err = r.q.UpdateDepositWithoutNote(ctx, sqlc.UpdateDepositWithoutNoteParams{
			Amount:      amount,
			DepositDate: depositDate,
			UpdatedAt:   now,
			ID:          id,
		})
	} else {
		row, err = r.q.UpdateDeposit(ctx, sqlc.UpdateDepositParams{
			Amount:      amount,
			DepositDate: depositDate,
			Note:        *note,
			UpdatedAt:   now,
			ID:          id,
		})
	}
	if err != nil {
		return models.Deposit{}, err
	}
	return toDeposit(row), nil
}

// Delete removes a deposit by ID.
func (r *DepositRepository) Delete(ctx context.Context, id uuidx.UUID) error {
	return r.q.DeleteDeposit(ctx, id)
}

// GetFirstDepositDate returns the earliest deposit_date, or nil if no deposits exist.
func (r *DepositRepository) GetFirstDepositDate(ctx context.Context) (*datex.Date, error) {
	d, err := r.q.GetFirstDepositDate(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func toDeposit(row sqlc.Deposit) models.Deposit {
	return models.Deposit{
		ID:          row.ID,
		Amount:      row.Amount,
		DepositDate: row.DepositDate,
		Note:        row.Note,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}
