// Package numeric provides a SQLite-compatible decimal type. Peewee's
// DecimalField(decimal_places=10, auto_round=False) stores values into a column
// with NUMERIC affinity: whole values come back as integers, fractional values
// as floats. This type round-trips both while exposing shopspring/decimal.Decimal
// for exact arithmetic in the domain layer.
package numeric

import (
	"database/sql/driver"
	"fmt"

	"github.com/shopspring/decimal"
)

// Decimal wraps shopspring/decimal.Decimal with SQLite NUMERIC compatible storage.
type Decimal struct {
	decimal.Decimal
}

// Zero is the decimal 0.
var Zero = Decimal{decimal.Zero}

// FromInt builds a Decimal from an int64.
func FromInt(i int64) Decimal { return Decimal{decimal.NewFromInt(i)} }

// FromString parses a decimal string exactly.
func FromString(s string) (Decimal, error) {
	d, err := decimal.NewFromString(s)
	if err != nil {
		return Decimal{}, err
	}
	return Decimal{d}, nil
}

// Wrap wraps a decimal.Decimal.
func Wrap(d decimal.Decimal) Decimal { return Decimal{d} }

// Scan implements sql.Scanner for the int64/float64/[]byte/string forms SQLite
// returns from a NUMERIC-affinity column.
func (d *Decimal) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		d.Decimal = decimal.Zero
		return nil
	case int64:
		d.Decimal = decimal.NewFromInt(v)
		return nil
	case float64:
		// Fractional values were already stored lossily as REAL by Peewee/SQLite;
		// NewFromFloat uses the shortest round-tripping representation, matching
		// Python's str(float).
		d.Decimal = decimal.NewFromFloat(v)
		return nil
	case []byte:
		return d.parse(string(v))
	case string:
		return d.parse(v)
	default:
		return fmt.Errorf("numeric: cannot scan %T into Decimal", src)
	}
}

func (d *Decimal) parse(s string) error {
	if s == "" {
		d.Decimal = decimal.Zero
		return nil
	}
	parsed, err := decimal.NewFromString(s)
	if err != nil {
		return fmt.Errorf("numeric: cannot parse decimal %q: %w", s, err)
	}
	d.Decimal = parsed
	return nil
}

// Value implements driver.Valuer. It emits an int64 for whole values and a
// decimal string otherwise, so SQLite's NUMERIC affinity stores integers as
// INTEGER (matching existing rows) and keeps fractional precision as text.
func (d Decimal) Value() (driver.Value, error) {
	if d.Decimal.IsInteger() {
		return d.Decimal.IntPart(), nil
	}
	return d.Decimal.String(), nil
}
