// Package stockformat provides stock display-name utilities, mirroring
// services/stock_name_formatter.py.
package stockformat

import "strings"

const etfSuffix = "증권상장지수투자신탁(주식)"

// FormatName strips the Korean ETF suffix from a stock display name.
func FormatName(name string) string {
	return strings.TrimSpace(strings.ReplaceAll(name, etfSuffix, ""))
}
