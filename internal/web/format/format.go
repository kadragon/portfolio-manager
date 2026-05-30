// Package format provides display helpers that mirror the Jinja2 filters and
// raw value rendering in the Python templates, so generated HTML matches.
package format

import (
	"strconv"
	"strings"
)

// Float mirrors Python's str(float): a decimal point is always present for
// finite values (e.g. 30.0 renders "30.0", not "30"), using the shortest
// round-tripping representation without an exponent for the value ranges used
// here (percentages 0–100). Used for raw {{ group.target_percentage }} output.
func Float(f float64) string {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	if !strings.Contains(s, ".") {
		s += ".0"
	}
	return s
}
