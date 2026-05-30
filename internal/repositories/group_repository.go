// Package repositories owns all database access (Golden Principle 1). Each
// repository wraps the sqlc-generated queries and maps rows to domain models.
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

// GroupRepository provides group CRUD operations.
type GroupRepository struct {
	q *sqlc.Queries
}

// NewGroupRepository builds a GroupRepository over the given queries handle.
func NewGroupRepository(q *sqlc.Queries) *GroupRepository {
	return &GroupRepository{q: q}
}

// ListAll returns every group in insertion (rowid) order, matching the Python
// repository's GroupModel.select() with no explicit ordering.
func (r *GroupRepository) ListAll(ctx context.Context) ([]models.Group, error) {
	rows, err := r.q.ListGroups(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.Group, 0, len(rows))
	for _, row := range rows {
		out = append(out, toGroup(row))
	}
	return out, nil
}

// Create inserts a new group and returns it.
func (r *GroupRepository) Create(ctx context.Context, name string, targetPercentage float64) (models.Group, error) {
	now := ktime.Now()
	row, err := r.q.CreateGroup(ctx, sqlc.CreateGroupParams{
		ID:               uuidx.New(),
		Name:             name,
		TargetPercentage: targetPercentage,
		CreatedAt:        now,
		UpdatedAt:        now,
	})
	if err != nil {
		return models.Group{}, err
	}
	return toGroup(row), nil
}

// GetByID returns the group with the given id, or nil if none exists (Python:
// get_or_none → None).
func (r *GroupRepository) GetByID(ctx context.Context, id uuidx.UUID) (*models.Group, error) {
	row, err := r.q.GetGroup(ctx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	g := toGroup(row)
	return &g, nil
}

// Update applies the provided fields (nil leaves a field unchanged), mirroring
// the Python repository's partial-update semantics, and refreshes updated_at
// (the BaseModel.save() hook).
func (r *GroupRepository) Update(ctx context.Context, id uuidx.UUID, name *string, targetPercentage *float64) (models.Group, error) {
	row, err := r.q.GetGroup(ctx, id)
	if err != nil {
		return models.Group{}, err
	}
	if name != nil {
		row.Name = *name
	}
	if targetPercentage != nil {
		row.TargetPercentage = *targetPercentage
	}
	updated, err := r.q.UpdateGroup(ctx, sqlc.UpdateGroupParams{
		Name:             row.Name,
		TargetPercentage: row.TargetPercentage,
		UpdatedAt:        ktime.Now(),
		ID:               id,
	})
	if err != nil {
		return models.Group{}, err
	}
	return toGroup(updated), nil
}

// Delete removes the group with the given id.
func (r *GroupRepository) Delete(ctx context.Context, id uuidx.UUID) error {
	return r.q.DeleteGroup(ctx, id)
}

func toGroup(row sqlc.Group) models.Group {
	return models.Group{
		ID:               row.ID,
		Name:             row.Name,
		TargetPercentage: row.TargetPercentage,
		CreatedAt:        row.CreatedAt.Time,
		UpdatedAt:        row.UpdatedAt.Time,
	}
}
