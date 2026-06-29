// Package templates provides shared helper functions for templ components.
package templates

import (
	"html"
	"strings"

	"github.com/kadragon/portfolio-manager/internal/accountformat"
	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/numeric"
	"github.com/kadragon/portfolio-manager/internal/stockformat"
	"github.com/kadragon/portfolio-manager/internal/web/format"
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

// rateColorClassForPtr returns a normal-surface CSS class for an optional rate.
func rateColorClassForPtr(rate *numeric.Decimal) string {
	if rate == nil {
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

// depositFormHTML renders the inline deposit edit form row (deposits/_form.html).
// The form spans multiple <td> cells — raw HTML required for the same reason as
// groupFormHTML.
func depositFormHTML(d models.Deposit) string {
	id := html.EscapeString(d.ID.String())
	date := html.EscapeString(d.DepositDate.ISO())
	amount := html.EscapeString(d.Amount.String())
	note := ""
	if d.Note.Valid {
		note = html.EscapeString(d.Note.String)
	}
	return `<tr id="deposit-` + id + `">
  <td>
    <form id="edit-deposit-` + id + `"
          hx-put="/deposits/` + id + `"
          hx-target="closest tr"
          hx-swap="outerHTML"
          data-request-message="입금 내역을 저장하는 중…">
      <label class="sr-only" for="deposit-date-` + id + `">입금 날짜</label>
      <input id="deposit-date-` + id + `" type="date" name="deposit_date" value="` + date + `" required autocomplete="off" class="input input-bordered input-sm">
  </td>
  <td>
      <label class="sr-only" for="deposit-amount-` + id + `">입금 금액</label>
      <input id="deposit-amount-` + id + `" type="number" name="amount" value="` + amount + `"
             min="0" step="1" required autocomplete="off" class="input input-bordered input-sm">
  </td>
  <td>
      <label class="sr-only" for="deposit-note-` + id + `">메모</label>
      <input id="deposit-note-` + id + `" type="text" name="note" value="` + note + `"
             placeholder="예: 메모 입력…" autocomplete="off" class="input input-bordered input-sm">
      <small class="text-xs text-base-content/50 mt-0.5">빈 값은 기존 메모 유지, <code class="text-xs bg-base-300 px-1.5 py-0.5 rounded">/clear</code> 입력 시 메모 삭제</small>
  </td>
  <td>
      <div class="flex flex-wrap gap-1.5">
        <button type="submit" class="btn btn-primary btn-sm">저장</button>
        <button type="button"
                class="btn btn-ghost btn-xs"
                hx-get="/deposits/` + id + `"
                hx-target="closest tr"
                hx-swap="outerHTML"
                data-request-message="수정 취소 중…">
          취소
        </button>
      </div>
    </form>
  </td>
</tr>`
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

// groupFormHTML renders the inline group edit row (groups/_form.html). That
// template intentionally opens a <form> in the first <td> and closes it in the
// third — valid for browsers (form owner spans cells) but not expressible in
// templ's well-formed element syntax, so it is emitted as raw HTML with the
// dynamic values escaped, preserving byte-level parity with the Python output.
func groupFormHTML(g models.Group) string {
	id := html.EscapeString(g.ID.String())
	name := html.EscapeString(g.Name)
	target := html.EscapeString(format.Float(g.TargetPercentage))
	return `<tr id="group-` + id + `">
  <td>
    <form id="edit-group-` + id + `"
          hx-put="/groups/` + id + `"
          hx-target="closest tr"
          hx-swap="outerHTML"
          data-request-message="그룹 정보를 저장하는 중…">
      <label class="sr-only" for="group-name-` + id + `">그룹 이름</label>
      <input id="group-name-` + id + `" type="text" name="name" value="` + name + `" required autocomplete="off"
             class="input input-bordered input-sm">
  </td>
  <td>
      <label class="sr-only" for="group-target-` + id + `">목표 비중</label>
      <input id="group-target-` + id + `" type="number" name="target_percentage" value="` + target + `"
             min="0" max="100" step="0.1" required autocomplete="off"
             class="input input-bordered input-sm">
  </td>
  <td>
      <div class="flex flex-wrap gap-1.5">
        <button type="submit" class="btn btn-primary btn-sm">저장</button>
        <button type="button"
                class="btn btn-ghost btn-xs"
                hx-get="/groups/` + id + `"
                hx-target="closest tr"
                hx-swap="outerHTML"
                data-request-message="수정 취소 중…">
          취소
        </button>
      </div>
    </form>
  </td>
</tr>`
}
