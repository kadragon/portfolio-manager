package accountformat_test

import (
	"testing"

	"github.com/kadragon/portfolio-manager/internal/accountformat"
	"github.com/kadragon/portfolio-manager/internal/numeric"
)

func mustDecimal(s string, t *testing.T) numeric.Decimal {
	t.Helper()
	d, err := numeric.FromString(s)
	if err != nil {
		t.Fatalf("numeric.FromString(%q): %v", s, err)
	}
	return d
}

func TestFormatKRW(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0", "₩0"},
		{"1234567", "₩1,234,567"},
		{"-5000", "-₩5,000"},
		{"1000000", "₩1,000,000"},
		{"999", "₩999"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := accountformat.FormatKRW(mustDecimal(tc.input, t))
			if got != tc.want {
				t.Errorf("FormatKRW(%s) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestFormatUSD(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0", "$0.00"},
		{"1234.56", "$1,234.56"},
		{"-100.5", "-$100.50"},
		{"1000000.99", "$1,000,000.99"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := accountformat.FormatUSD(mustDecimal(tc.input, t))
			if got != tc.want {
				t.Errorf("FormatUSD(%s) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestFormatPercent(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0", "0.0%"},
		{"12.3", "12.3%"},
		{"-5.5", "-5.5%"},
		{"100", "100.0%"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := accountformat.FormatPercent(mustDecimal(tc.input, t))
			if got != tc.want {
				t.Errorf("FormatPercent(%s) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestFormatSignedPercent(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0", "0.0%"},
		{"12.3", "+12.3%"},
		{"-5.5", "-5.5%"},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := accountformat.FormatSignedPercent(mustDecimal(tc.input, t))
			if got != tc.want {
				t.Errorf("FormatSignedPercent(%s) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
