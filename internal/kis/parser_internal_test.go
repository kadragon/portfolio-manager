package kis

import (
	"testing"

	"github.com/shopspring/decimal"
)

func TestParseFloatEdge(t *testing.T) {
	cases := []struct {
		in   string
		want float64
	}{
		{"", 0},
		{"  ", 0},
		{"abc", 0},
		{"123", 123},
		{" 45.67 ", 45.67},
		{"-3.5", -3.5},
	}
	for _, c := range cases {
		if got := parseFloat(c.in); got != c.want {
			t.Errorf("parseFloat(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestExtractFirstOutputForms(t *testing.T) {
	// object form
	obj := extractFirstOutput([]byte(`{"output":{"a":"1"}}`))
	if obj["a"] != "1" {
		t.Errorf("object form: a = %q, want 1", obj["a"])
	}
	// array form (takes index 0)
	arr := extractFirstOutput([]byte(`{"output":[{"b":"2"},{"b":"3"}]}`))
	if arr["b"] != "2" {
		t.Errorf("array form: b = %q, want 2", arr["b"])
	}
	// missing output key
	if m := extractFirstOutput([]byte(`{"other":1}`)); len(m) != 0 {
		t.Errorf("missing output: len = %d, want 0", len(m))
	}
	// invalid JSON
	if m := extractFirstOutput([]byte(`not json`)); len(m) != 0 {
		t.Errorf("invalid JSON: len = %d, want 0", len(m))
	}
	// empty array
	if m := extractFirstOutput([]byte(`{"output":[]}`)); len(m) != 0 {
		t.Errorf("empty array: len = %d, want 0", len(m))
	}
}

func TestParseDecimalEdge(t *testing.T) {
	cases := []struct {
		in   string
		want decimal.Decimal
	}{
		{"", decimal.Zero},
		{"   ", decimal.Zero},
		{"bad", decimal.Zero},
		{"1234", decimal.NewFromInt(1234)},
		{" 12.5 ", decimal.NewFromFloat(12.5)},
	}
	for _, c := range cases {
		if got := parseDecimal(c.in); !got.Equal(c.want) {
			t.Errorf("parseDecimal(%q) = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestStrVal(t *testing.T) {
	m := map[string]any{
		"s":   "hello",
		"n":   42,
		"nil": nil,
	}
	if got := strVal(m, "s"); got != "hello" {
		t.Errorf("strVal(s) = %q, want hello", got)
	}
	if got := strVal(m, "n"); got != "42" {
		t.Errorf("strVal(n) = %q, want 42", got)
	}
	if got := strVal(m, "nil"); got != "" {
		t.Errorf("strVal(nil) = %q, want empty", got)
	}
	if got := strVal(m, "missing"); got != "" {
		t.Errorf("strVal(missing) = %q, want empty", got)
	}
}

func TestToSliceOfMaps(t *testing.T) {
	if toSliceOfMaps(nil) != nil {
		t.Errorf("toSliceOfMaps(nil) should be nil")
	}
	// single map → one-element slice
	single := toSliceOfMaps(map[string]any{"k": "v"})
	if len(single) != 1 || single[0]["k"] != "v" {
		t.Errorf("toSliceOfMaps(map) = %v, want [{k:v}]", single)
	}
	// slice of maps → filtered slice (skips non-map entries)
	multi := toSliceOfMaps([]any{
		map[string]any{"x": 1},
		"not a map",
		map[string]any{"y": 2},
	})
	if len(multi) != 2 {
		t.Errorf("toSliceOfMaps([]any) len = %d, want 2", len(multi))
	}
	// unsupported type → nil
	if toSliceOfMaps(42) != nil {
		t.Errorf("toSliceOfMaps(int) should be nil")
	}
}
