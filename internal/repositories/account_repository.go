package repositories

import (
	"context"
	"database/sql"
	"errors"

	"github.com/kadragon/portfolio-manager/internal/db/sqlc"
	"github.com/kadragon/portfolio-manager/internal/ktime"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/uuidx"
)

// AccountRepository provides account CRUD operations.
type AccountRepository struct {
	q *sqlc.Queries
}

// NewAccountRepository builds an AccountRepository over the given queries handle.
func NewAccountRepository(q *sqlc.Queries) *AccountRepository {
	return &AccountRepository{q: q}
}

// Create inserts a new account with zero cash balance by default.
func (r *AccountRepository) Create(ctx context.Context, name string, cashBalance numeric.Decimal) (models.Account, error) {
	now := ktime.Now()
	row, err := r.q.CreateAccount(ctx, sqlc.CreateAccountParams{
		ID:          uuidx.New(),
		Name:        name,
		CashBalance: cashBalance,
		CreatedAt:   now,
		UpdatedAt:   now,
	})
	if err != nil {
		return models.Account{}, err
	}
	return toAccount(row), nil
}

// ListAll returns all accounts in insertion order.
func (r *AccountRepository) ListAll(ctx context.Context) ([]models.Account, error) {
	rows, err := r.q.ListAccounts(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.Account, 0, len(rows))
	for _, row := range rows {
		out = append(out, toAccount(row))
	}
	return out, nil
}

// GetByID returns the account with the given id, or nil if absent.
func (r *AccountRepository) GetByID(ctx context.Context, id uuidx.UUID) (*models.Account, error) {
	row, err := r.q.GetAccountByID(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	a := toAccount(row)
	return &a, nil
}

// UpdateNameCash updates only name and cash_balance (used by bulk-cash; preserves KIS fields).
func (r *AccountRepository) UpdateNameCash(ctx context.Context, id uuidx.UUID, name string, cashBalance numeric.Decimal) (models.Account, error) {
	row, err := r.q.UpdateAccountNameCash(ctx, sqlc.UpdateAccountNameCashParams{
		Name:        name,
		CashBalance: cashBalance,
		UpdatedAt:   ktime.Now(),
		ID:          id,
	})
	if err != nil {
		return models.Account{}, err
	}
	return toAccount(row), nil
}

// Update sets all editable fields including broker bindings and account type.
func (r *AccountRepository) Update(
	ctx context.Context,
	id uuidx.UUID,
	name string,
	cashBalance numeric.Decimal,
	kisAccountNo sql.NullString,
	kisAPIKeyID sql.NullInt64,
	accountType sql.NullString,
	tossAccountSeq sql.NullInt64,
) (models.Account, error) {
	row, err := r.q.UpdateAccount(ctx, sqlc.UpdateAccountParams{
		Name:           name,
		CashBalance:    cashBalance,
		KisAccountNo:   kisAccountNo,
		KisApiKeyID:    kisAPIKeyID,
		AccountType:    accountType,
		TossAccountSeq: tossAccountSeq,
		UpdatedAt:      ktime.Now(),
		ID:             id,
	})
	if err != nil {
		return models.Account{}, err
	}
	return toAccount(row), nil
}

// DeleteWithHoldings removes holdings then the account (two sequential statements, parity with Python).
func (r *AccountRepository) DeleteWithHoldings(ctx context.Context, id uuidx.UUID) error {
	if err := r.q.DeleteHoldingsByAccount(ctx, id); err != nil {
		return err
	}
	return r.q.DeleteAccount(ctx, id)
}

func toAccount(row sqlc.Account) models.Account {
	var kisAccountNo *string
	if row.KisAccountNo.Valid {
		kisAccountNo = &row.KisAccountNo.String
	}
	var kisAPIKeyID *int64
	if row.KisApiKeyID.Valid {
		kisAPIKeyID = &row.KisApiKeyID.Int64
	}
	var accountType *string
	if row.AccountType.Valid {
		accountType = &row.AccountType.String
	}
	var tossAccountSeq *int64
	if row.TossAccountSeq.Valid {
		tossAccountSeq = &row.TossAccountSeq.Int64
	}
	return models.Account{
		ID:             row.ID,
		Name:           row.Name,
		CashBalance:    row.CashBalance,
		CreatedAt:      row.CreatedAt.Time,
		UpdatedAt:      row.UpdatedAt.Time,
		KisAccountNo:   kisAccountNo,
		KisAPIKeyID:    kisAPIKeyID,
		AccountType:    accountType,
		TossAccountSeq: tossAccountSeq,
	}
}
