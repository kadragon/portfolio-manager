// Package accountformat formats account monetary values for display.
package accountformat

import (
	"strings"

	"github.com/kadragon/portfolio-manager/internal/numeric"
)

// FormatKRW formats a decimal as ₩1,234,567 (0 decimal places, comma thousands).
// Matches Python's f"₩{d:,.0f}".
func FormatKRW(d numeric.Decimal) string {
	rounded := d.Round(0)
	s := rounded.StringFixed(0)
	neg := len(s) > 0 && s[0] == '-'
	if neg {
		s = s[1:]
	}
	var b strings.Builder
	n := len(s)
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	result := "₩" + b.String()
	if neg {
		return "-" + result
	}
	return result
}

// FormatUSD formats a decimal as $1,234.56 (2 decimal places, comma thousands).
// Matches Python's f"${d:,.2f}".
func FormatUSD(d numeric.Decimal) string {
	rounded := d.StringFixed(2)
	parts := strings.SplitN(rounded, ".", 2)
	intPart := parts[0]
	fracPart := ""
	if len(parts) == 2 {
		fracPart = parts[1]
	}
	neg := len(intPart) > 0 && intPart[0] == '-'
	if neg {
		intPart = intPart[1:]
	}
	var b strings.Builder
	n := len(intPart)
	for i, c := range intPart {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(c)
	}
	result := "$" + b.String() + "." + fracPart
	if neg {
		return "-" + result
	}
	return result
}

// FormatPercent formats a decimal as "12.3%" (1 decimal place, half-away-from-zero).
// Matches Python's d.quantize(Decimal("0.1"), rounding=ROUND_HALF_UP).
func FormatPercent(d numeric.Decimal) string {
	return d.StringFixed(1) + "%"
}

// FormatSignedPercent formats a decimal as "+12.3%" or "-12.3%".
// Matches Python's format_signed_percent filter.
func FormatSignedPercent(d numeric.Decimal) string {
	s := d.StringFixed(1) + "%"
	if d.IsPositive() {
		return "+" + s
	}
	return s
}
