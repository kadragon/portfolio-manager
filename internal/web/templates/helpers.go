package templates

import (
	"html"

	"github.com/kadragon/portfolio-manager/internal/models"
	"github.com/kadragon/portfolio-manager/internal/web/format"
)

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
	{"/insights", "insights", "AI 인사이트"},
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
