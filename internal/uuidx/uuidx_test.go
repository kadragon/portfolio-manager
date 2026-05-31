package uuidx

import (
	"testing"

	"github.com/google/uuid"
)

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

func TestNew(t *testing.T) {
	a, b := New(), New()
	if a == b {
		t.Error("New() returned duplicate UUIDs")
	}
	if a.UUID == uuid.Nil {
		t.Error("New() returned nil UUID")
	}
}

func TestWrap(t *testing.T) {
	id := uuid.New()
	w := Wrap(id)
	if w.UUID != id {
		t.Errorf("Wrap() = %v, want %v", w.UUID, id)
	}
}

func TestParseError(t *testing.T) {
	_, err := Parse("not-a-uuid")
	if err == nil {
		t.Error("Parse invalid expected error, got nil")
	}
}

func TestScanBytesUUID(t *testing.T) {
	id := New()
	var u UUID
	if err := u.Scan([]byte(id.Hex())); err != nil {
		t.Fatalf("Scan []byte: %v", err)
	}
	if u.UUID != id.UUID {
		t.Errorf("Scan result mismatch")
	}
}

func TestScanUnknownTypeUUID(t *testing.T) {
	var u UUID
	if err := u.Scan(42); err == nil {
		t.Error("Scan int expected error")
	}
}

func TestParseEmptyString(t *testing.T) {
	var u UUID
	if err := u.Scan(""); err != nil {
		t.Fatalf("Scan empty string: %v", err)
	}
	if u.UUID != uuid.Nil {
		t.Errorf("empty string should give nil UUID, got %v", u.UUID)
	}
}

func TestParseInvalidString(t *testing.T) {
	var u UUID
	if err := u.Scan("not-valid-hex"); err == nil {
		t.Error("Scan invalid expected error")
	}
}
