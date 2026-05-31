// Package datex provides a SQLite-compatible date-only type matching Peewee's
// DateField, which stores ISO "YYYY-MM-DD" TEXT.
package datex

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/kadragon/portfolio-manager/internal/ktime"
)

const layout = "2006-01-02"

// Date wraps a time.Time but serializes as an ISO date string.
type Date struct {
	time.Time
}

// New builds a Date from year, month, day in KST.
func New(year int, month time.Month, day int) Date {
	return Date{time.Date(year, month, day, 0, 0, 0, 0, ktime.KST)}
}

// FromTime truncates a time.Time to its date in KST.
func FromTime(t time.Time) Date {
	t = t.In(ktime.KST)
	return Date{time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, ktime.KST)}
}

// ParseDate parses an ISO "YYYY-MM-DD" string.
func ParseDate(s string) (Date, error) {
	var d Date
	if err := d.parse(s); err != nil {
		return Date{}, err
	}
	return d, nil
}

// ISO returns the "YYYY-MM-DD" representation.
func (d Date) ISO() string {
	if d.Time.IsZero() {
		return ""
	}
	return d.Time.Format(layout)
}

// Scan implements sql.Scanner for the TEXT/time forms SQLite returns.
func (d *Date) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		d.Time = time.Time{}
		return nil
	case time.Time:
		d.Time = v
		return nil
	case []byte:
		return d.parse(string(v))
	case string:
		return d.parse(v)
	default:
		return fmt.Errorf("datex: cannot scan %T into Date", src)
	}
}

func (d *Date) parse(s string) error {
	if s == "" {
		d.Time = time.Time{}
		return nil
	}
	// Tolerate a stored datetime by taking its date prefix.
	if len(s) > len(layout) {
		s = s[:len(layout)]
	}
	parsed, err := time.ParseInLocation(layout, s, ktime.KST)
	if err != nil {
		return fmt.Errorf("datex: cannot parse date %q: %w", s, err)
	}
	d.Time = parsed
	return nil
}

// Value implements driver.Valuer, writing the ISO date TEXT form.
func (d Date) Value() (driver.Value, error) {
	if d.Time.IsZero() {
		return nil, nil
	}
	return d.ISO(), nil
}
