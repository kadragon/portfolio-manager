<!--
Schema / lifecycle:
  Active sprint  — open [ ] items grouped by PR or feature branch below this header.
  Dormant        — no active sprint; file starts with "# (dormant)" and retains only open [debt/blocked] items.
-->

# (dormant) Tasks — Deferred from PR reviews

## Review Backlog

### PR #100 — [HARNESS] Upgrade harness to Level 2 (2026-05-27)

- [ ] [doc] `docs/delegation.md` Model Selection table — pin model versions or add note to resolve from `~/.claude/settings.json`; bare tier names (sonnet/opus) may drift on model upgrade (source: pr-review-toolkit:review-pr) — `docs/delegation.md:52`
- [ ] [doc] `docs/adr/README.md` ADR template missing `## Options Considered` section — AGENTS.md says ADRs document "options considered" but template has only Context/Decision/Consequences (source: pr-review-toolkit:review-pr) — `docs/adr/README.md`
- [ ] [doc] `docs/eval-criteria.md` Correctness criterion "How to test: Run the feature manually" — replace with specific `go test` command or scenario list for CI-reproducibility (note: original item said "pytest command"; repo is now Go) (source: review) — `docs/eval-criteria.md:37`

### PR #111 — [FIX] remove KIS key ID value from fallback log (2026-06-02)

- [ ] [constraint] `resolveSyncService` has no unit tests — add table-driven test covering: nil keyID, found key, key=1 not found (no log), key≠1 not found (warning log path) (source: pr-review-toolkit:review-pr) — `internal/container/container.go:225`

### PR #112 — [FEAT] resolve historical prices to nearest prior trading day (2026-06-03)

- [ ] [perf] `GetPortfolioSummary` fetches `GetUSDKRW()` eagerly to populate the display rate, adding one cold EXIM lookup for KRW-only portfolios — bounded to 1 fetch/day by `cachedRates`, but consider decoupling the display-rate fetch from valuation if it shows up in latency (source: codex) — `internal/services/portfolio_service.go:133`

### PR #113 — [FEAT] tax-aware rebalancing (2026-06-03)

- [x] [debt] target allocation per-account split uses `accountAUM.Div(typeCap)` with no remainder absorption — sub-won FP drift when AUM ratios don't divide evenly. Conservation tests (`TestPlannerTaxLocation`) pass with exact `.Equal`, so drift is below material scale; deferred. Fix = assign remainder to final account (order-dependent absorber, weigh trade-off) (source: agy) — `internal/services/rebalance_service.go:508` — **obsolete: `accountAUM.Div(typeCap)` per-account AUM split removed in PR #117 (commit `d31e028`); engine now allocates per group via `allocateSells`/`buildBuyRecs`, no AUM split remains.**
- [x] [debt] negative account AUM (huge negative cash) yields negative target values; clamp AUM to zero before splitting. Defensive against theoretical state, no failing test (source: agy) — `internal/services/rebalance_service.go:494` — **obsolete: same per-account AUM split removed in PR #117 (commit `d31e028`); no AUM-derived target values remain to clamp.**
- [x] [test] no guard that every `_groupOrder` entry has a `_placementScore` key — silent default-0 score if a group is added without updating the score map (source: pr-review-toolkit:review-pr) — `internal/services/rebalance_service.go:379` — **resolved: `TestPlacementScoreCoversAllGroups` in `planner_test.go` asserts every `_groupOrder` entry has a `_placementScore` row.**
- [ ] [refactor] `assetClassEquals` two-line helper used once — inline for parity with `accounts.go` (source: pr-review-toolkit:review-pr) — `internal/web/handlers/stocks.go:24`
- [ ] [refactor] add `models.ValidAssetClass(s)` mirroring `ValidAccountType` to centralize "etf"/"stock" vocabulary (source: pr-review-toolkit:review-pr) — `internal/models/account.go`

## Review Backlog

### PR #114 — KIS ETF classification + tax-location rebalance reasoning (2026-06-03)

- [ ] [debt] ETN (scty_grp_id_cd "EN") classified as "stock" blocks IRP/연금 buys; verify KR ETN eligibility for 연금/IRP and model if eligible (source: pr-review-toolkit:review-pr) — internal/kis/domestic_info.go:34, internal/services/rebalance_service.go canHold
- [ ] [debt] Unclassifiable/failed tickers keep asset_class=nil and are re-queried on every sync/ClassifyAll; persist an "unknown" sentinel (needs schema + edit-form value decision) to stop redundant KIS calls (source: agy) — internal/services/stock_classification.go:27
- [ ] [debt] ClassifyAll loops KIS calls synchronously with no throttle inside the web handler; large unclassified sets risk KIS rate-limit and HTTP timeout — add inter-call delay or background job + HTMX polling (source: agy) — internal/services/stock_classification.go:88
- [ ] [doc] Drop redundant html.EscapeString in classifyStocks/syncAccount handlers (templ auto-escapes `{ message }`, output is double-escaped) — cosmetic, currently consistent with sibling handler (source: security-review) — internal/web/handlers/accounts.go:272

### PR #117 — dead-code cleanup + review backlog (2026-06-04)

- [ ] [debt] `security_group` update handler accepts any free-text (uppercased/trimmed) while `asset_class` enforces an allowlist; an allowlist here must match every code KIS sync legitimately writes (ST/EF/EN/EW/MF/RT/FE/FS + any unseen) or it breaks sync — needs canonical code-set decision before guarding (source: pr-review-toolkit:review-pr) — internal/web/handlers/stocks.go:209
- [ ] [perf] `calcQuantity`/`krwToLocal` (pre-existing) divide before multiply; decimal.Div truncates, so reorder Mul-before-Div to cut precision loss — behavior-changing on pinned test expectations, needs careful test (source: agy) — internal/services/rebalance_service.go calcQuantity/krwToLocal
- [ ] [test] no KIS_LIVE-guarded integration test that a real overseas KIS response round-trips through `OverseasSecurityGroup` (FE/FS); unit-tested only (source: pr-review-toolkit:review-pr) — internal/kis/overseas_info.go

### PR #119 — refactor(ui): separate page canvas from card surface (2026-06-04)

- [ ] [harness] `internal/web/static/css/app.css` is tracked in git; compiled output creates noisy diffs and build-env divergence risk — add to `.gitignore` and generate in CI/Docker instead (source: pr-review-toolkit:review-pr) — `internal/web/static/css/app.css`
- [ ] [debt] DaisyUI base-100/200 role inversion from convention (100=card, 200=canvas) is intentional but confusing; document design decision in `docs/` or DESIGN.md comment block (source: review) — `internal/web/tailwind/input.css:15-19`
- [ ] [doc] Commit message format: project uses `[TYPE]` prefix (AGENTS.md), not Conventional Commits `type(scope):` — align or update `docs/conventions.md` to accept both (source: pr-review-toolkit:review-pr)
