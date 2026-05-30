package datex

import "testing"

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
