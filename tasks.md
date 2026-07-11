<!--
Schema / lifecycle:
  Active sprint  — open [ ] items grouped by PR or feature branch below this header.
  Dormant        — no active sprint; file starts with "# (dormant)" and retains only open [debt/blocked] items.
-->

# Tasks — Deferred from PR reviews

## Review Backlog

### PR #114 — KIS ETF classification + tax-location rebalance reasoning (2026-06-03)

- [ ] [debt] ETN (scty_grp_id_cd "EN") classified as "stock" blocks IRP/연금 buys; verify KR ETN eligibility for 연금/IRP and model if eligible (source: pr-review-toolkit:review-pr) — internal/kis/domestic_info.go:34, internal/services/rebalance_service.go canHold
- [x] [debt] Unclassifiable/failed tickers keep asset_class=nil and are re-queried on every sync/ClassifyAll; persist an "unknown" sentinel (needs schema + edit-form value decision) to stop redundant KIS calls (source: agy) — internal/services/stock_classification.go:27 — **resolved: `AssetClassUnknown` sentinel stamped on asset_class ONLY (security_group keeps KIS code space); `isUnknown(asset_class)` terminal for all skip-gates (no schema change — TEXT col); edit form "" resets to re-query; sentinel set by classifier only, not client POST.**
- [x] [debt] ClassifyAll loops KIS calls synchronously with no throttle inside the web handler; large unclassified sets risk KIS rate-limit and HTTP timeout — add inter-call delay or background job + HTMX polling (source: agy) — internal/services/stock_classification.go:88 — **resolved: ctx-aware inter-call delay (`SetCallDelay`, container injects 200ms) + loop-top ctx.Err() guard; background-job+HTMX-polling deferred (see Out of scope).**
- [x] [doc] Drop redundant html.EscapeString in classifyStocks/syncAccount handlers (templ auto-escapes `{ message }`, output is double-escaped) — cosmetic, currently consistent with sibling handler (source: security-review) — internal/web/handlers/accounts.go:272 — **obsolete: internal/web/ removed (web UI teardown, cmd/pm CLI replaces it).**

### PR #119 — refactor(ui): separate page canvas from card surface (2026-06-04)

- [x] [harness] `internal/web/static/css/app.css` is tracked in git; compiled output creates noisy diffs and build-env divergence risk — add to `.gitignore` and generate in CI/Docker instead (source: pr-review-toolkit:review-pr) — `internal/web/static/css/app.css` — **obsolete: internal/web/, Tailwind toolchain, and Docker removed (web UI teardown).**
- [x] [debt] DaisyUI base-100/200 role inversion from convention (100=card, 200=canvas) is intentional but confusing; document design decision in `docs/` or DESIGN.md comment block (source: review) — `internal/web/tailwind/input.css:15-19` — **obsolete: internal/web/ and Tailwind toolchain removed (web UI teardown).**
- [x] [doc] Commit message format: project uses `[TYPE]` prefix (AGENTS.md), not Conventional Commits `type(scope):` — align or update `docs/conventions.md` to accept both (source: pr-review-toolkit:review-pr) — **resolved: `docs/conventions.md:5` already documents `[TYPE]` format, explicitly "Conventional Commits 사용 안 함".**

### PR #129 — [REFACTOR] centralize asset-class vocabulary via models.ValidAssetClass (2026-06-19)

- [ ] [refactor] `AssetClassUnknown = "unknown"` sentinel lives in `services` while the new valid-class consts (`AssetClassETF`/`AssetClassStock`) live in `models`; co-locating the sentinel in `models/stock.go` would unify the asset_class value space, but ripples to external `services.AssetClassUnknown` references in test files (out of PR #129 scope) (source: review) — `internal/services/stock_classification.go:20`

### PR #132 — fix/toss-post-review-fixes (2026-06-28)

- [ ] [debt] `accountOrderRouter.PlaceOrder` and `ExecuteRebalanceOrders` persistence loop use `context.Background()` — `OrderClient` interface carries no `ctx` parameter (same pattern as existing KIS order clients). Systemic fix: add `ctx context.Context` to `OrderClient.PlaceOrder`, `kisOrderPlacer`, `tossOrderPlacer` interfaces and all implementations (`kis.DomesticOrderClient`, `kis.OverseasOrderClient`, `toss.Client.PlaceOrder`) (source: agy, review) — `internal/container/container.go:258`, `internal/services/rebalance_execution_service.go:170`

### PR #135 — [FEAT] skip buy recs when executable qty < 1 whole share (2026-06-29)

- [ ] [debt] `hasExecutableWholeShare` applies a blanket ≥1 whole-share floor to all currencies including USD, where some brokers support fractional shares; if fractional trading is ever enabled for overseas stocks, this guard will need per-currency / per-account-type gating (source: agy) — `internal/services/rebalance_service.go:961`

### PR #137 — PR #117 backlog close (2026-06-30)

- [ ] [debt] `ValidSecurityGroup` allowlist may reject unknown-but-legitimate future KIS `scty_grp_id_cd` codes not in the initial 8; classifier path intentionally bypasses it, but extend the allowlist if KIS adds new codes (source: codex, review) — `internal/models/stock.go:ValidSecurityGroup`

### PR #136 — [FEAT] add dashboard benchmark comparison (2026-06-30)

- [ ] [debt] `computeBenchmarkReturns` skips showing benchmark return rates when `portfolioReturn == nil` (shows "-" for all values); benchmark rates could be shown without the diff column when portfolio return is unavailable — design decision deferred (source: inline) — `internal/services/portfolio_service.go`
- [ ] [perf] `syncHistoricalDates` appends `firstDepositDate` to the shared list used for all sync targets; only benchmark tickers need this date for `GetStockChangeSince`. Fix: split into base dates (all targets) and benchmark-only dates (source: open-code-review) — `internal/services/price_sync_service.go:204`
- [ ] [debt] `computeBenchmarkAverage` returns a partial average when fewer than all benchmarks have a `ReturnRate`; template shows "평균" with no indication of partial coverage. Fix: add `BenchmarkAvailableCount` to `PortfolioSummary` and reflect in template (source: open-code-review) — `internal/services/portfolio_service.go:310`

### PR #145 — [FEAT] add cmd/pm CLI, replace web UI with CLI + skills (2026-07-11)

- [ ] [debt] `pm sync -account NAME` silently falls back to the `.env` default `KIS_CANO`/`KIS_ACNT_PRDT_CD` when the account has no `kis_account_no` linked, overwriting the account's local holdings/cash with a different (default) KIS account's snapshot instead of erroring — pre-existing behavior ported unchanged from the removed `AccountHandler.syncAccount` (same fallback existed at `internal/web/handlers/accounts.go:250-254` on `main`), not introduced by this PR, but worth hardening now that it's exposed via a scriptable CLI (source: agy) — `cmd/pm/sync.go:75-84`
- [ ] [debt] `pm account update -kis-account-no ""` (clearing the KIS account number) does not also clear `-kis-api-key-id`, leaving a stale `kis_api_key_id` set with no `kis_account_no` — pre-existing coupling gap ported unchanged from the removed `AccountHandler.update` (same gap on `main` at `internal/web/handlers/accounts.go:166-181`), not introduced by this PR (source: review) — `cmd/pm/account.go:123-129`
