package datex

import (
	"testing"
	"time"
)

func TestScanISOStringRoundTrip(t *testing.T) {
	// Form stored by Peewee DateField.
	const stored = "2021-01-06"
	var d Date
	if err := d.Scan(stored); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if d.ISO() != stored {
		t.Fatalf("iso = %q, want %q", d.ISO(), stored)
	}
	v, err := d.Value()
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	if s, ok := v.(string); !ok || s != stored {
		t.Fatalf("value = %v (%T), want %q", v, v, stored)
	}
}

func TestScanBytes(t *testing.T) {
	var d Date
	if err := d.Scan([]byte("2026-01-08")); err != nil {
		t.Fatalf("scan bytes: %v", err)
	}
	if d.ISO() != "2026-01-08" {
		t.Fatalf("iso = %q", d.ISO())
	}
}

func TestScanDatetimePrefix(t *testing.T) {
	var d Date
	if err := d.Scan("2026-03-28 13:21:44.873677+00:00"); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if d.ISO() != "2026-03-28" {
		t.Fatalf("iso = %q, want 2026-03-28", d.ISO())
	}
}

func TestScanNil(t *testing.T) {
	var d Date
	if err := d.Scan(nil); err != nil {
		t.Fatalf("scan nil: %v", err)
	}
	v, err := d.Value()
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	if v != nil {
		t.Fatalf("zero date value = %v, want nil", v)
	}
}

func TestNew(t *testing.T) {
	d := New(2026, 1, 6)
	if d.ISO() != "2026-01-06" {
		t.Fatalf("iso = %q, want 2026-01-06", d.ISO())
	}
}

func TestFromTime(t *testing.T) {
	tt := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	d := FromTime(tt)
	if d.ISO() != "2026-01-03" {
		t.Errorf("FromTime = %q, want 2026-01-03", d.ISO())
	}
}

func TestParseDate(t *testing.T) {
	d, err := ParseDate("2026-01-03")
	if err != nil {
		t.Fatalf("ParseDate: %v", err)
	}
	if d.ISO() != "2026-01-03" {
		t.Errorf("ParseDate = %q, want 2026-01-03", d.ISO())
	}
	_, err = ParseDate("not-a-date")
	if err == nil {
		t.Error("ParseDate(invalid) should return error")
	}
}

func TestISOZero(t *testing.T) {
	var d Date
	if d.ISO() != "" {
		t.Errorf("zero Date.ISO() = %q, want empty", d.ISO())
	}
}

func TestScanTimeTime(t *testing.T) {
	var d Date
	tt := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	if err := d.Scan(tt); err != nil {
		t.Fatalf("Scan time.Time: %v", err)
	}
	if d.ISO() != "2026-01-03" {
		t.Errorf("Scan time.Time = %q, want 2026-01-03", d.ISO())
	}
}

func TestScanBytesDate(t *testing.T) {
	var d Date
	if err := d.Scan([]byte("2026-05-01")); err != nil {
		t.Fatalf("Scan []byte: %v", err)
	}
	if d.ISO() != "2026-05-01" {
		t.Errorf("Scan []byte = %q, want 2026-05-01", d.ISO())
	}
}

func TestScanUnknownType(t *testing.T) {
	var d Date
	err := d.Scan(42)
	if err == nil {
		t.Error("Scan int expected error, got nil")
	}
}
