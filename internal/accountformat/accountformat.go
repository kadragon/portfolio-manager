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
