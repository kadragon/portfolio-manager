package numeric

import "testing"

func TestScanInt64WholeValue(t *testing.T) {
	// Whole decimals come back from SQLite NUMERIC affinity as int64.
	var d Decimal
	if err := d.Scan(int64(968616)); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if d.String() != "968616" {
		t.Fatalf("string = %q, want %q", d.String(), "968616")
	}
	v, err := d.Value()
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	// Whole values must write back as INTEGER to match existing rows.
	if i, ok := v.(int64); !ok || i != 968616 {
		t.Fatalf("value = %v (%T), want int64 968616", v, v)
	}
}

func TestScanFloatFractionalRoundTrip(t *testing.T) {
	// Fractional decimals come back as float64; the literals below are the
	// exact fractional quantities present in the production DB.
	cases := map[float64]string{
		0.94:      "0.94",
		12.677005: "12.677005",
		9.5:       "9.5",
		0.123456:  "0.123456",
		8.5:       "8.5",
	}
	for in, want := range cases {
		var d Decimal
		if err := d.Scan(in); err != nil {
			t.Fatalf("scan %v: %v", in, err)
		}
		if d.String() != want {
			t.Fatalf("scan %v -> %q, want %q", in, d.String(), want)
		}
		v, err := d.Value()
		if err != nil {
			t.Fatalf("value %v: %v", in, err)
		}
		if s, ok := v.(string); !ok || s != want {
			t.Fatalf("value %v = %v (%T), want string %q", in, v, v, want)
		}
	}
}

func TestScanStringExact(t *testing.T) {
	var d Decimal
	if err := d.Scan("12.677005"); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if d.String() != "12.677005" {
		t.Fatalf("string = %q", d.String())
	}
}

func TestScanNilZero(t *testing.T) {
	var d Decimal
	if err := d.Scan(nil); err != nil {
		t.Fatalf("scan nil: %v", err)
	}
	if !d.Decimal.IsZero() {
		t.Fatalf("nil -> %q, want 0", d.String())
	}
	v, err := d.Value()
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	if i, ok := v.(int64); !ok || i != 0 {
		t.Fatalf("zero value = %v (%T), want int64 0", v, v)
	}
}
