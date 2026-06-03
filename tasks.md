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

- [ ] [debt] target allocation per-account split uses `accountAUM.Div(typeCap)` with no remainder absorption — sub-won FP drift when AUM ratios don't divide evenly. Conservation tests (`TestPlannerTaxLocation`) pass with exact `.Equal`, so drift is below material scale; deferred. Fix = assign remainder to final account (order-dependent absorber, weigh trade-off) (source: agy) — `internal/services/rebalance_service.go:508`
- [ ] [debt] negative account AUM (huge negative cash) yields negative target values; clamp AUM to zero before splitting. Defensive against theoretical state, no failing test (source: agy) — `internal/services/rebalance_service.go:494`
- [ ] [test] no guard that every `_groupOrder` entry has a `_placementScore` key — silent default-0 score if a group is added without updating the score map (source: pr-review-toolkit:review-pr) — `internal/services/rebalance_service.go:379`
- [ ] [refactor] `assetClassEquals` two-line helper used once — inline for parity with `accounts.go` (source: pr-review-toolkit:review-pr) — `internal/web/handlers/stocks.go:24`
- [ ] [refactor] add `models.ValidAssetClass(s)` mirroring `ValidAccountType` to centralize "etf"/"stock" vocabulary (source: pr-review-toolkit:review-pr) — `internal/models/account.go`
