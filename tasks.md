<!--
Schema / lifecycle:
  Active sprint  — open [ ] items grouped by PR or feature branch below this header.
  Dormant        — no active sprint; file starts with "# (dormant)" and retains only open [debt/blocked] items.
-->

# Tasks — Deferred from PR reviews

## Review Backlog

### PR #129 — [REFACTOR] centralize asset-class vocabulary via models.ValidAssetClass (2026-06-19)

- [ ] [refactor] `AssetClassUnknown = "unknown"` sentinel lives in `services` while the new valid-class consts (`AssetClassETF`/`AssetClassStock`) live in `models`; co-locating the sentinel in `models/stock.go` would unify the asset_class value space, but ripples to external `services.AssetClassUnknown` references in test files (out of PR #129 scope) (source: review) — `internal/services/stock_classification.go:20`

### PR #164 — [FIX] omit change-rate keys with no cached history (2026-08-01)

- [ ] [test] The `GetStockChangeRates` omit-contract and the `sortDashboardHoldings` present-before-missing rule are each pinned only in isolation — `./cmd/pm` passes in full even with the `price_service.go` omit hunk reverted, so nothing exercises `runDashboard` end-to-end on a ticker with genuinely short history. An integration test over a seeded DB (short-history ticker + `-sort 1y`) would close the gap between the two halves (source: qa-verifier) — `cmd/pm/dashboard.go:sortDashboardHoldings`, `internal/services/price_service.go:107`

### PR #149 — [FIX] thread ctx.Context through KIS/Toss OrderClient.PlaceOrder (2026-07-12)

- [ ] [debt] `toss.Client.PlaceOrder`'s `ctx` isn't forwarded into `c.accessToken()`'s token-refresh HTTP call, so a cancelled/timed-out caller ctx doesn't abort an in-flight Toss OAuth request — it still blocks up to the HTTP client's default 30s timeout. Fixing requires giving `accessToken` a `ctx` param, which is also called by `FetchAccountSnapshot` (no `ctx` param today); deferred to keep this PR scoped to the order-placement path only (source: agy) — `internal/toss/client.go:139`

### PR #150 — [DOCS] document ETN pension-account ineligibility as intentional classification (2026-07-12)

- [ ] [debt] `ClassifyDomesticAssetClass` classifies ETN via a two-step fallback (scty_grp_id_cd not "EF"/"FE", then etf_dvsn_cd empty/"0" → "stock"); no explicit `grp == "EN"` guard exists, so a KIS response where an ETN unexpectedly carries a non-empty/non-"0" etf_dvsn_cd would misclassify it as "etf" — pre-existing logic, not introduced by this PR, but worth an explicit guard if ever observed in practice (source: agy) — `internal/kis/domestic_info.go:ClassifyDomesticAssetClass`

### PR #151 — [FIX] validate -security-group against KIS allowlist in pm stock update (2026-07-14)

- [ ] [debt] `ValidSecurityGroup` allowlist may reject unknown-but-legitimate future KIS `scty_grp_id_cd` codes not in the initial 8; this PR restores CLI-side enforcement of the existing allowlist but does not extend the allowlist itself, so the underlying concern from PR #137 is still unaddressed — extend the allowlist if KIS adds new codes (source: review, contest-round) — `internal/models/stock.go:ValidSecurityGroup`

### PR #157 — [REFACTOR] migrate rebalance snapshot.py to pm snapshot subcommand (2026-07-16)

- [ ] [debt] `pm snapshot` boots the shared `container.New("")`, which opens the DB read-write and applies pending schema migrations before dispatch — the removed `snapshot.py` opened SQLite `mode=ro`, so on a missing/pre-migration DB the new command can create/migrate production state where the old one could not. This is a pre-existing property of every `pm` subcommand (dashboard, account, …), not introduced by this PR, but a read-only container path for read-only commands like `snapshot`/`dashboard` would restore the "planning does not update the DB" guarantee (source: codex) — `cmd/pm/main.go:54`, `internal/container/container.go`
- [ ] [debt] `pm snapshot` rounds with shopspring `.Round(0)`/`.Round(2)` (half-away-from-zero), whereas the removed `snapshot.py` used Python `round()` (banker's rounding). Real-DB output is currently byte-identical, but a value whose rounded digit sits exactly on `.5` (plausible for USD-converted amounts and 2-dp weights) would differ by 1 unit — a latent parity break. Half-away is arguably the better money rounding; decide whether exact snapshot.py parity matters and, if so, switch to `RoundBank` for value_krw/totals/weight_pct (source: review, confidence 48/low) — `cmd/pm/snapshot.go:205`
