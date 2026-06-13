// Package templates provides shared helper functions for templ components.
package templates

import (
	"html"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/accountformat"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/stockformat"
)

// formatQty formats a holding quantity by market, keyed off ticker length:
// domestic tickers are exactly 6-character KRX codes (cf. kis.IsDomesticTicker)
// and show an integer with thousands separators; everything else is an overseas
// ticker (e.g. "AAPL", "GOOGL") and shows one decimal place to preserve
// fractional shares.
func formatQty(ticker string, qty numeric.Decimal) string {
	if len(ticker) != 6 {
		return qty.StringFixed(1)
	}
	s := qty.Round(0).StringFixed(0)
	neg := strings.HasPrefix(s, "-")
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
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// formatRate formats the USD/KRW exchange rate for display, or "-" when absent.
func formatRate(rate *numeric.Decimal) string {
	if rate == nil || !rate.IsPositive() {
		return "-"
	}
	return accountformat.FormatKRW(*rate)
}

// rateColorClassForMap returns a CSS class for a change-rate value. Returns "" if period absent.
func rateColorClassForMap(rates map[string]numeric.Decimal, period string) string {
	rate, ok := rates[period]
	if !ok {
		return ""
	}
	if rate.IsPositive() {
		return "text-success"
	}
	if rate.IsNegative() {
		return "text-error"
	}
	return ""
}

// rateColorClassForDark returns dark-theme CSS class for a change-rate value.
func rateColorClassForDark(rate *numeric.Decimal) string {
	if rate == nil {
		return ""
	}
	if rate.IsPositive() {
		return "text-up-on-dark"
	}
	if rate.IsNegative() {
		return "text-down-on-dark"
	}
	return ""
}

// signedRateHTML returns an HTML snippet: formatted signed percent, or the dim dash span.
// safe to use with templ.Raw().
func signedRateHTML(rates map[string]numeric.Decimal, period string) string {
	rate, ok := rates[period]
	if !ok {
		return `<span class="text-base-content/30">-</span>`
	}
	return html.EscapeString(accountformat.FormatSignedPercent(rate))
}

// diffColorClass returns a CSS class for a diff value (positive=error, negative=success).
func diffColorClass(d numeric.Decimal) string {
	if d.IsPositive() {
		return "text-error"
	}
	if d.IsNegative() {
		return "text-success"
	}
	return ""
}

// stockName returns the formatted name if non-empty, else ticker.
func stockName(name, ticker string) string {
	if formatted := stockformat.FormatName(name); formatted != "" {
		return formatted
	}
	return ticker
}

// noteValue returns the deposit's stored note for prefilling the edit field, or
// "" when unset.
func noteValue(d models.Deposit) string {
	if d.Note.Valid {
		return d.Note.String
	}
	return ""
}

// navItem is one entry in the top navigation bar (base.html nav_items).
type navItem struct {
	Href  string
	Page  string
	Label string
}

// navItems mirrors the hardcoded navbar list in base.html.
var navItems = []navItem{
	{"/", "dashboard", "대시보드"},
	{"/groups", "groups", "그룹"},
	{"/accounts", "accounts", "계좌"},
	{"/deposits", "deposits", "입금"},
	{"/rebalance", "rebalance", "리밸런싱"},
}

// navClass returns the nav link class string (base.html), active or not.
func navClass(active bool) string {
	const base = "rounded-full px-3.5 py-2 text-[13px] font-medium transition-colors "
	if active {
		return base + "bg-primary text-primary-content hover:bg-primary"
	}
	return base + "text-base-content/60 hover:bg-base-200 hover:text-base-content"
}
