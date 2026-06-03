// Package db opens and initializes the SQLite database, mirroring the behavior
// of services/database.py: WAL journal mode, foreign-key enforcement, the
// PORTFOLIO_DB_PATH override, and the default .data/portfolio.db location.
package db

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // registers the sqlite3 driver

	"github.com/kadragon/portfolio-manager/internal/db/sqlc"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the embedded DDL (used by tests and tooling).
func Schema() string { return schemaSQL }

// addedColumns are columns introduced by the Go layer after the original
// Peewee schema. On a fresh database schema.sql already creates them; on an
// existing production database the CREATE TABLE IF NOT EXISTS is a no-op, so
// migrate() appends them with ALTER TABLE. Always nullable TEXT (SQLite forbids
// ALTER ADD COLUMN NOT NULL without a default on a non-empty table).
var addedColumns = []struct{ table, column string }{
	{"stocks", "asset_class"},
	{"accounts", "account_type"},
	{"stocks", "security_group"},
}

// migrate applies idempotent ALTER TABLE ADD COLUMN for every entry in
// addedColumns that is not already present. Safe to run on every Open.
func migrate(ctx context.Context, db *sql.DB) error {
	for _, ac := range addedColumns {
		has, err := hasColumn(ctx, db, ac.table, ac.column)
		if err != nil {
			return err
		}
		if has {
			continue
		}
		stmt := fmt.Sprintf("ALTER TABLE %q ADD COLUMN %q TEXT", ac.table, ac.column)
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("db: add column %s.%s: %w", ac.table, ac.column, err)
		}
	}
	return nil
}

// hasColumn reports whether table already has the named column.
func hasColumn(ctx context.Context, db *sql.DB, table, column string) (bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%q)", table))
	if err != nil {
		return false, fmt.Errorf("db: table_info(%s): %w", table, err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var (
			cid     int
			name    string
			ctype   string
			notnull int
			dflt    sql.NullString
			pk      int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, fmt.Errorf("db: scan table_info(%s): %w", table, err)
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

// DefaultPath resolves the database path the same way the Python app does:
// PORTFOLIO_DB_PATH if set, otherwise <projectRoot>/.data/portfolio.db.
func DefaultPath() (string, error) {
	if env := os.Getenv("PORTFOLIO_DB_PATH"); env != "" {
		return filepath.Abs(env)
	}
	root, err := projectRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".data", "portfolio.db"), nil
}

// projectRoot walks up from the working directory looking for go.mod.
func projectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("db: cannot locate project root (go.mod); set PORTFOLIO_DB_PATH")
		}
		dir = parent
	}
}

// dsn builds a modernc.org/sqlite DSN with WAL and foreign keys enabled.
func dsn(path string) string {
	return "file:" + path + "?_pragma=journal_mode(wal)&_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)"
}

// Open opens the database at path (or the default if empty), creates tables if
// needed, and returns the *sql.DB together with the sqlc Queries handle.
func Open(path string) (*sql.DB, *sqlc.Queries, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, nil, err
		}
	}
	if path != ":memory:" {
		// path is operator-controlled config (PORTFOLIO_DB_PATH or the computed
		// project-local default), never untrusted request input.
		if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
			return nil, nil, fmt.Errorf("db: mkdir: %w", err)
		}
	}
	sqlDB, err := sql.Open("sqlite", dsn(path))
	if err != nil {
		return nil, nil, fmt.Errorf("db: open: %w", err)
	}
	// SQLite with WAL tolerates one writer; a single connection avoids
	// "database is locked" under concurrent writes from the web layer.
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.ExecContext(context.Background(), schemaSQL); err != nil {
		_ = sqlDB.Close()
		return nil, nil, fmt.Errorf("db: create tables: %w", err)
	}
	if err := migrate(context.Background(), sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, nil, err
	}
	return sqlDB, sqlc.New(sqlDB), nil
}

// OpenMemory opens a fresh in-memory database with the schema applied, for tests.
func OpenMemory() (*sql.DB, *sqlc.Queries, error) {
	sqlDB, err := sql.Open("sqlite", "file::memory:?_pragma=foreign_keys(1)&cache=shared")
	if err != nil {
		return nil, nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.ExecContext(context.Background(), schemaSQL); err != nil {
		_ = sqlDB.Close()
		return nil, nil, err
	}
	if err := migrate(context.Background(), sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, nil, err
	}
	return sqlDB, sqlc.New(sqlDB), nil
}
