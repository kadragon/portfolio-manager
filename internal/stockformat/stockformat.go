// Package stockformat provides stock display-name utilities, mirroring
// services/stock_name_formatter.py.
package stockformat

import "strings"

const etfSuffixMarker = "증권상장지수투자신탁"

// FormatName strips the Korean ETF suffix from a stock display name.
// Handles truncated variants (e.g. "...신탁(" from KIS 40-char prdt_name limit)
// and non-standard types (e.g. "(주식-파생형)") by cutting at the marker.
func FormatName(name string) string {
	if i := strings.Index(name, etfSuffixMarker); i >= 0 {
		return strings.TrimSpace(name[:i])
	}
	return name
}
