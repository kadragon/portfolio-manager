// Package ktime provides the project-wide KST (Asia/Seoul) timezone and a
// SQLite-compatible timestamp type. KST is the project-wide timezone, matching
// the Python core/time.py module.
package ktime

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// KST is the Asia/Seoul location, the project-wide timezone.
var KST = mustLoadKST()

func mustLoadKST() *time.Location {
	loc, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		// Fallback to a fixed +09:00 offset if the tz database is unavailable.
		return time.FixedZone("KST", 9*60*60)
	}
	return loc
}

// NowKST returns the current time in KST, mirroring Python's now_kst().
func NowKST() time.Time {
	return time.Now().In(KST)
}

// writeLayout always emits 6 microsecond digits and a colon offset, matching
// Python's str() of a timezone-aware datetime produced by datetime.now(KST):
// "2026-01-03 13:21:44.873677+09:00" (space separator, numeric offset).
const writeLayout = "2006-01-02 15:04:05.000000-07:00"

// parseLayouts lists accepted stored representations, most specific first.
// Existing rows were written with a "+00:00" offset; new rows use "+09:00".
// Some legacy rows may lack microseconds or an offset entirely.
var parseLayouts = []string{
	"2006-01-02 15:04:05.999999-07:00",
	"2006-01-02 15:04:05-07:00",
	"2006-01-02 15:04:05.999999",
	"2006-01-02 15:04:05",
	time.RFC3339Nano,
	time.RFC3339,
}

// Time wraps time.Time with SQLite (Peewee DateTimeField) compatible
// serialization. Stored as TEXT.
type Time struct {
	time.Time
}

// Now returns the current KST time wrapped as a Time.
func Now() Time { return Time{NowKST()} }

// New wraps a time.Time.
func New(t time.Time) Time { return Time{t} }

// Scan implements sql.Scanner, parsing the TEXT/time representations SQLite returns.
func (t *Time) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		t.Time = time.Time{}
		return nil
	case time.Time:
		t.Time = v
		return nil
	case []byte:
		return t.parse(string(v))
	case string:
		return t.parse(v)
	default:
		return fmt.Errorf("ktime: cannot scan %T into Time", src)
	}
}

func (t *Time) parse(s string) error {
	if s == "" {
		t.Time = time.Time{}
		return nil
	}
	for _, l := range parseLayouts {
		if parsed, err := time.ParseInLocation(l, s, KST); err == nil {
			t.Time = parsed
			return nil
		}
	}
	return fmt.Errorf("ktime: cannot parse timestamp %q", s)
}

// Value implements driver.Valuer, emitting the Python str(datetime) format.
func (t Time) Value() (driver.Value, error) {
	if t.Time.IsZero() {
		return nil, nil
	}
	return t.Time.Format(writeLayout), nil
}

// String returns the stored representation.
func (t Time) String() string {
	if t.Time.IsZero() {
		return ""
	}
	return t.Time.Format(writeLayout)
}
