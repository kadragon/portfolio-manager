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
)

// Container holds shared dependencies for the web layer.
type Container struct {
	DB     *sql.DB
	Groups *repositories.GroupRepository
	Stocks *repositories.StockRepository
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
	return &Container{
		DB:     sqlDB,
		Groups: repositories.NewGroupRepository(q),
		Stocks: repositories.NewStockRepository(q),
	}
}

// Close releases the database connection.
func (c *Container) Close() error {
	if c.DB != nil {
		return c.DB.Close()
	}
	return nil
}
