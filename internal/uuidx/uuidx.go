// Package uuidx provides a SQLite-compatible UUID type. Peewee's UUIDField
// stores the 32-character lowercase hex form (no dashes); this type reads and
// writes that exact representation while exposing the canonical google/uuid.UUID.
package uuidx

import (
	"database/sql/driver"
	"fmt"

	"github.com/google/uuid"
)

// UUID wraps google/uuid.UUID with Peewee-compatible (hex32, no dashes) storage.
type UUID struct {
	uuid.UUID
}

// New returns a random UUID (v4), matching Python's uuid4 default.
func New() UUID { return UUID{uuid.New()} }

// Wrap wraps a uuid.UUID.
func Wrap(u uuid.UUID) UUID { return UUID{u} }

// Parse accepts both the dashed canonical form and the 32-char hex form.
func Parse(s string) (UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return UUID{}, err
	}
	return UUID{u}, nil
}

// Hex returns the 32-character lowercase hex form without dashes, the on-disk format.
func (u UUID) Hex() string {
	b := u.UUID
	return fmt.Sprintf("%x", b[:])
}

// Scan implements sql.Scanner for the stored hex32 (or dashed) TEXT form.
func (u *UUID) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		u.UUID = uuid.Nil
		return nil
	case []byte:
		return u.parse(string(v))
	case string:
		return u.parse(v)
	default:
		return fmt.Errorf("uuidx: cannot scan %T into UUID", src)
	}
}

func (u *UUID) parse(s string) error {
	if s == "" {
		u.UUID = uuid.Nil
		return nil
	}
	parsed, err := uuid.Parse(s)
	if err != nil {
		return fmt.Errorf("uuidx: cannot parse UUID %q: %w", s, err)
	}
	u.UUID = parsed
	return nil
}

// Value implements driver.Valuer, writing the 32-char hex form Peewee uses.
func (u UUID) Value() (driver.Value, error) {
	if u.UUID == uuid.Nil {
		return nil, nil
	}
	return u.Hex(), nil
}
