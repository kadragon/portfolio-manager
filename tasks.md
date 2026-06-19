<!--
Schema / lifecycle:
  Active sprint  — open [ ] items grouped by PR or feature branch below this header.
  Dormant        — no active sprint; file starts with "# (dormant)" and retains only open [debt/blocked] items.
-->

# Tasks — Deferred from PR reviews

## Sprint Contract — stock classification sentinel + throttle

status: done

**Scope:** PR #114 review-backlog debt — (1) persist an "unknown" sentinel for
unclassifiable stocks so failed/overseas tickers stop being re-queried via KIS on every
sync/`ClassifyAll`; (2) throttle the synchronous `ClassifyAll` KIS loop. Files:
`internal/services/stock_classification.go`, `internal/container/container.go`,
`internal/web/handlers/stocks.go`, `internal/web/templates/stocks.templ` (+ regen `_templ.go`),
`internal/services/stock_classification_test.go`.

**Acceptance criteria:**
- [x] Classifier failure/empty-signal on a nil-asset_class stock persists `AssetClassUnknown`
      on asset_class ONLY (security_group keeps its KIS value space); `isUnknown(asset_class)`
      is terminal for all three skip-gates (ClassifyAll, classifyStock, account-sync) so the
      ticker stops being re-queried even with a nil security_group. No-signal counts as Failed.
- [x] Partially-classified stock (asset_class set, security_group nil) is NOT sentinel-stamped
      (existing `TestStockClassificationServiceClassifyAll` stays green).
- [x] `ClassifyAll` honors ctx cancellation (loop-top guard + ctx-aware inter-call delay);
      delay injected by container (`SetCallDelay(200ms)`), 0 in tests.
- [x] Edit form round-trips "unknown" (allowlist + templ label "분류 실패"); "" still clears → re-query.
- [x] `go test ./...`, arch test, golangci-lint, `templ generate --check` all green.

**Out of scope:** background-job + HTMX-polling for ClassifyAll (deferred follow-up);
`models.ValidAssetClass` centralization (separate backlog item).

**Lint/test:** `go test ./... && golangci-lint run && templ generate --check`

## Sprint Contract — lazy display-rate fetch (PR #112 follow-up)

status: done

**Scope:** PR #112 review-backlog perf finding — `GetPortfolioSummary` eagerly calls
`GetUSDKRW()` before the holdings loop, forcing one cold EXIM lookup even for KRW-only
portfolios. Defer the fetch so it fires only when a USD holding is actually valued.
File: `internal/services/portfolio_service.go` (+ `internal/services/portfolio_service_test.go`).

**Acceptance criteria:**
- [x] KRW-only portfolio triggers 0 `FetchUSDRate` calls (lazy fetch on first USD holding).
- [x] USD portfolio still values correctly and `USDKRWRate` is populated.
- [x] `go test ./... && golangci-lint run && templ generate --check` all green.

**Out of scope:** ExchangeRateService caching changes; display-rate UI behavior.
**Decision:** KRW-only portfolio now renders dashboard USD/KRW as `-` (no USD exposure → no
fetch); user confirmed hiding the rate is preferred over keeping the cold lookup.

**Lint/test:** `go test ./... && golangci-lint run && templ generate --check`

## Review Backlog

### PR #100 — [HARNESS] Upgrade harness to Level 2 (2026-05-27)

- [x] [doc] `docs/delegation.md` Model Selection table — pin model versions or add note to resolve from `~/.claude/settings.json`; bare tier names (sonnet/opus) may drift on model upgrade (source: pr-review-toolkit:review-pr) — `docs/delegation.md:52` — **resolved: note already present at `docs/delegation.md:58`.**
- [ ] [doc] `docs/adr/README.md` ADR template missing `## Options Considered` section — AGENTS.md says ADRs document "options considered" but template has only Context/Decision/Consequences (source: pr-review-toolkit:review-pr) — `docs/adr/README.md`
- [x] [doc] `docs/eval-criteria.md` Correctness criterion "How to test: Run the feature manually" — replace with specific `go test` command or scenario list for CI-reproducibility (note: original item said "pytest command"; repo is now Go) (source: review) — `docs/eval-criteria.md:37` — **resolved: `docs/eval-criteria.md:37` already reads `go test ./...` green + manual verify.**

### PR #111 — [FIX] remove KIS key ID value from fallback log (2026-06-02)

- [x] [constraint] `resolveSyncService` has no unit tests — add table-driven test covering: nil keyID, found key, key=1 not found (no log), key≠1 not found (warning log path) (source: pr-review-toolkit:review-pr) — `internal/container/container.go:225` — **resolved: `TestResolveSyncService` in `container_test.go` covers all 4 cases + log-capture assertions.**

### PR #112 — [FEAT] resolve historical prices to nearest prior trading day (2026-06-03)

- [x] [perf] `GetPortfolioSummary` fetches `GetUSDKRW()` eagerly to populate the display rate, adding one cold EXIM lookup for KRW-only portfolios — bounded to 1 fetch/day by `cachedRates`, but consider decoupling the display-rate fetch from valuation if it shows up in latency (source: codex) — `internal/services/portfolio_service.go:133` — **resolved: USD/KRW now fetched lazily via `resolveUSDKRW` closure on first USD holding (memoizes nil); KRW-only portfolios make 0 EXIM calls and render the dashboard rate as `-`. Tests `TestGetPortfolioSummaryKRWOnlyNoRateFetch`/`...USDFetchesRate`.**

### PR #113 — [FEAT] tax-aware rebalancing (2026-06-03)

- [x] [debt] target allocation per-account split uses `accountAUM.Div(typeCap)` with no remainder absorption — sub-won FP drift when AUM ratios don't divide evenly. Conservation tests (`TestPlannerTaxLocation`) pass with exact `.Equal`, so drift is below material scale; deferred. Fix = assign remainder to final account (order-dependent absorber, weigh trade-off) (source: agy) — `internal/services/rebalance_service.go:508` — **obsolete: `accountAUM.Div(typeCap)` per-account AUM split removed in PR #117 (commit `d31e028`); engine now allocates per group via `allocateSells`/`buildBuyRecs`, no AUM split remains.**
- [x] [debt] negative account AUM (huge negative cash) yields negative target values; clamp AUM to zero before splitting. Defensive against theoretical state, no failing test (source: agy) — `internal/services/rebalance_service.go:494` — **obsolete: same per-account AUM split removed in PR #117 (commit `d31e028`); no AUM-derived target values remain to clamp.**
- [x] [test] no guard that every `_groupOrder` entry has a `_placementScore` key — silent default-0 score if a group is added without updating the score map (source: pr-review-toolkit:review-pr) — `internal/services/rebalance_service.go:379` — **resolved: `TestPlacementScoreCoversAllGroups` in `planner_test.go` asserts every `_groupOrder` entry has a `_placementScore` row.**
- [ ] [refactor] `assetClassEquals` two-line helper used once — inline for parity with `accounts.go` (source: pr-review-toolkit:review-pr) — `internal/web/handlers/stocks.go:24`
- [x] [refactor] add `models.ValidAssetClass(s)` mirroring `ValidAccountType` to centralize "etf"/"stock" vocabulary (source: pr-review-toolkit:review-pr) — `internal/models/account.go` — **resolved: `ValidAssetClass` + `AssetClassETF`/`AssetClassStock` consts added to `internal/models/stock.go`; call sites in `web/handlers/stocks.go` + `services/stock_classification.go` swapped to it; `TestValidAssetClass` table test added.**

## Review Backlog

### PR #114 — KIS ETF classification + tax-location rebalance reasoning (2026-06-03)

- [ ] [debt] ETN (scty_grp_id_cd "EN") classified as "stock" blocks IRP/연금 buys; verify KR ETN eligibility for 연금/IRP and model if eligible (source: pr-review-toolkit:review-pr) — internal/kis/domestic_info.go:34, internal/services/rebalance_service.go canHold
- [x] [debt] Unclassifiable/failed tickers keep asset_class=nil and are re-queried on every sync/ClassifyAll; persist an "unknown" sentinel (needs schema + edit-form value decision) to stop redundant KIS calls (source: agy) — internal/services/stock_classification.go:27 — **resolved: `AssetClassUnknown` sentinel stamped on asset_class ONLY (security_group keeps KIS code space); `isUnknown(asset_class)` terminal for all skip-gates (no schema change — TEXT col); edit form "" resets to re-query; sentinel set by classifier only, not client POST.**
- [x] [debt] ClassifyAll loops KIS calls synchronously with no throttle inside the web handler; large unclassified sets risk KIS rate-limit and HTTP timeout — add inter-call delay or background job + HTMX polling (source: agy) — internal/services/stock_classification.go:88 — **resolved: ctx-aware inter-call delay (`SetCallDelay`, container injects 200ms) + loop-top ctx.Err() guard; background-job+HTMX-polling deferred (see Out of scope).**
- [ ] [doc] Drop redundant html.EscapeString in classifyStocks/syncAccount handlers (templ auto-escapes `{ message }`, output is double-escaped) — cosmetic, currently consistent with sibling handler (source: security-review) — internal/web/handlers/accounts.go:272

### PR #117 — dead-code cleanup + review backlog (2026-06-04)

- [ ] [debt] `security_group` update handler accepts any free-text (uppercased/trimmed) while `asset_class` enforces an allowlist; an allowlist here must match every code KIS sync legitimately writes (ST/EF/EN/EW/MF/RT/FE/FS + any unseen) or it breaks sync — needs canonical code-set decision before guarding (source: pr-review-toolkit:review-pr) — internal/web/handlers/stocks.go:209
- [ ] [perf] `calcQuantity`/`krwToLocal` (pre-existing) divide before multiply; decimal.Div truncates, so reorder Mul-before-Div to cut precision loss — behavior-changing on pinned test expectations, needs careful test (source: agy) — internal/services/rebalance_service.go calcQuantity/krwToLocal
- [ ] [test] no KIS_LIVE-guarded integration test that a real overseas KIS response round-trips through `OverseasSecurityGroup` (FE/FS); unit-tested only (source: pr-review-toolkit:review-pr) — internal/kis/overseas_info.go

### PR #119 — refactor(ui): separate page canvas from card surface (2026-06-04)

- [ ] [harness] `internal/web/static/css/app.css` is tracked in git; compiled output creates noisy diffs and build-env divergence risk — add to `.gitignore` and generate in CI/Docker instead (source: pr-review-toolkit:review-pr) — `internal/web/static/css/app.css`
- [ ] [debt] DaisyUI base-100/200 role inversion from convention (100=card, 200=canvas) is intentional but confusing; document design decision in `docs/` or DESIGN.md comment block (source: review) — `internal/web/tailwind/input.css:15-19`
- [x] [doc] Commit message format: project uses `[TYPE]` prefix (AGENTS.md), not Conventional Commits `type(scope):` — align or update `docs/conventions.md` to accept both (source: pr-review-toolkit:review-pr) — **resolved: `docs/conventions.md:5` already documents `[TYPE]` format, explicitly "Conventional Commits 사용 안 함".**

### PR #122 — [DOCS] fix review-backlog doc findings (2026-06-19)

- [x] [doc] `docs/conventions.md:34` + `docs/architecture.md:23` claim `//go:build integration` build tag for (KIS) integration tests, but no `.go` file uses it — real gate is `t.Skip`+`KIS_LIVE=1` (AGENTS.md GP-2). Reconcile both untouched docs to the actual mechanism (source: review) — `docs/conventions.md:34`, `docs/architecture.md:23` — **resolved: both docs now describe `KIS_LIVE=1` + `t.Skip` gate (GP-2); confirmed zero `.go` files use `//go:build integration`.**

### PR #129 — [REFACTOR] centralize asset-class vocabulary via models.ValidAssetClass (2026-06-19)

- [ ] [refactor] `AssetClassUnknown = "unknown"` sentinel lives in `services` while the new valid-class consts (`AssetClassETF`/`AssetClassStock`) live in `models`; co-locating the sentinel in `models/stock.go` would unify the asset_class value space, but ripples to external `services.AssetClassUnknown` references in test files (out of PR #129 scope) (source: review) — `internal/services/stock_classification.go:20`
