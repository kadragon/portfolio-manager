package uuidx

import "testing"

func TestParseHex32RoundTrip(t *testing.T) {
	// Form stored by Peewee UUIDField (32 hex chars, no dashes).
	const stored = "5aa9c13b1ac74c0dabe2a9ee715b0f84"
	var u UUID
	if err := u.Scan(stored); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if got := u.Hex(); got != stored {
		t.Fatalf("hex = %q, want %q", got, stored)
	}
	v, err := u.Value()
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	if v.(string) != stored {
		t.Fatalf("value = %v, want %q", v, stored)
	}
}

func TestParseDashedAcceptedStoredAsHex(t *testing.T) {
	const dashed = "5aa9c13b-1ac7-4c0d-abe2-a9ee715b0f84"
	const hex = "5aa9c13b1ac74c0dabe2a9ee715b0f84"
	u, err := Parse(dashed)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if u.Hex() != hex {
		t.Fatalf("hex = %q, want %q", u.Hex(), hex)
	}
}

func TestScanNilAndEmpty(t *testing.T) {
	var u UUID
	if err := u.Scan(nil); err != nil {
		t.Fatalf("scan nil: %v", err)
	}
	v, err := u.Value()
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	if v != nil {
		t.Fatalf("nil uuid value = %v, want nil", v)
	}
}

func TestScanBytes(t *testing.T) {
	const stored = "5aa9c13b1ac74c0dabe2a9ee715b0f84"
	var u UUID
	if err := u.Scan([]byte(stored)); err != nil {
		t.Fatalf("scan bytes: %v", err)
	}
	if u.Hex() != stored {
		t.Fatalf("hex = %q, want %q", u.Hex(), stored)
	}
}
