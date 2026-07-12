<!--
Schema / lifecycle:
  Active sprint  — open [ ] items grouped by PR or feature branch below this header.
  Dormant        — no active sprint; file starts with "# (dormant)" and retains only open [debt/blocked] items.
-->

# Tasks — Deferred from PR reviews

## Review Backlog

### PR #129 — [REFACTOR] centralize asset-class vocabulary via models.ValidAssetClass (2026-06-19)

- [ ] [refactor] `AssetClassUnknown = "unknown"` sentinel lives in `services` while the new valid-class consts (`AssetClassETF`/`AssetClassStock`) live in `models`; co-locating the sentinel in `models/stock.go` would unify the asset_class value space, but ripples to external `services.AssetClassUnknown` references in test files (out of PR #129 scope) (source: review) — `internal/services/stock_classification.go:20`

### PR #135 — [FEAT] skip buy recs when executable qty < 1 whole share (2026-06-29)

- [ ] [debt] `hasExecutableWholeShare` applies a blanket ≥1 whole-share floor to all currencies including USD, where some brokers support fractional shares; if fractional trading is ever enabled for overseas stocks, this guard will need per-currency / per-account-type gating (source: agy) — `internal/services/rebalance_service.go:961`

### PR #136 — [FEAT] add dashboard benchmark comparison (2026-06-30)

- [ ] [debt] `computeBenchmarkReturns` skips showing benchmark return rates when `portfolioReturn == nil` (shows "-" for all values); benchmark rates could be shown without the diff column when portfolio return is unavailable — design decision deferred (source: inline) — `internal/services/portfolio_service.go`
- [ ] [perf] `syncHistoricalDates` appends `firstDepositDate` to the shared list used for all sync targets; only benchmark tickers need this date for `GetStockChangeSince`. Fix: split into base dates (all targets) and benchmark-only dates (source: open-code-review) — `internal/services/price_sync_service.go:204`
- [ ] [debt] `computeBenchmarkAverage` returns a partial average when fewer than all benchmarks have a `ReturnRate`; template shows "평균" with no indication of partial coverage. Fix: add `BenchmarkAvailableCount` to `PortfolioSummary` and reflect in template (source: open-code-review) — `internal/services/portfolio_service.go:310`

### PR #145 — [FEAT] add cmd/pm CLI, replace web UI with CLI + skills (2026-07-11)

- [ ] [debt] `pm sync -account NAME` silently falls back to the `.env` default `KIS_CANO`/`KIS_ACNT_PRDT_CD` when the account has no `kis_account_no` linked, overwriting the account's local holdings/cash with a different (default) KIS account's snapshot instead of erroring — pre-existing behavior ported unchanged from the removed `AccountHandler.syncAccount` (same fallback existed at `internal/web/handlers/accounts.go:250-254` on `main`), not introduced by this PR, but worth hardening now that it's exposed via a scriptable CLI (source: agy) — `cmd/pm/sync.go:75-84`
- [ ] [debt] `pm account update -kis-account-no ""` (clearing the KIS account number) does not also clear `-kis-api-key-id`, leaving a stale `kis_api_key_id` set with no `kis_account_no` — pre-existing coupling gap ported unchanged from the removed `AccountHandler.update` (same gap on `main` at `internal/web/handlers/accounts.go:166-181`), not introduced by this PR (source: review) — `cmd/pm/account.go:123-129`

### PR #147 — [FEAT] add pm price list command and dashboard -sort flag (2026-07-11)

- [ ] [debt] `GetStockChangeRates` intentionally stores a literal `0` (not an absent key) when a stock lacks cached history far enough back for a period (see `TestGetStockChangeRatesZeroForMissingHistory`); `pm dashboard -sort` can't distinguish that from a genuine flat return, so a recently-added ticker sorts as if flat rather than excluded/last. Fixing properly requires changing `GetStockChangeRates`'s return contract (e.g. a parallel "has data" map or `*Decimal`), which ripples to its one caller and existing tests — architectural change beyond this PR's scope (source: codex) — `internal/services/price_service.go:104`, `cmd/pm/dashboard.go:73`

### PR #149 — [FIX] thread ctx.Context through KIS/Toss OrderClient.PlaceOrder (2026-07-12)

- [ ] [debt] `toss.Client.PlaceOrder`'s `ctx` isn't forwarded into `c.accessToken()`'s token-refresh HTTP call, so a cancelled/timed-out caller ctx doesn't abort an in-flight Toss OAuth request — it still blocks up to the HTTP client's default 30s timeout. Fixing requires giving `accessToken` a `ctx` param, which is also called by `FetchAccountSnapshot` (no `ctx` param today); deferred to keep this PR scoped to the order-placement path only (source: agy) — `internal/toss/client.go:139`

### PR #150 — [DOCS] document ETN pension-account ineligibility as intentional classification (2026-07-12)

- [ ] [debt] `ClassifyDomesticAssetClass` classifies ETN via a two-step fallback (scty_grp_id_cd not "EF"/"FE", then etf_dvsn_cd empty/"0" → "stock"); no explicit `grp == "EN"` guard exists, so a KIS response where an ETN unexpectedly carries a non-empty/non-"0" etf_dvsn_cd would misclassify it as "etf" — pre-existing logic, not introduced by this PR, but worth an explicit guard if ever observed in practice (source: agy) — `internal/kis/domestic_info.go:ClassifyDomesticAssetClass`
