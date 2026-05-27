<!--
Schema / lifecycle:
  Active sprint  — open [ ] items grouped by PR or feature branch below this header.
  Dormant        — no active sprint; file starts with "# (dormant)" and retains only open [debt/blocked] items.
-->

# Tasks — Deferred from PR reviews

## Review Backlog

### PR #100 — [HARNESS] Upgrade harness to Level 2 (2026-05-27)

- [ ] [doc] `docs/delegation.md` Model Selection table — pin model versions or add note to resolve from `~/.claude/settings.json`; bare tier names (sonnet/opus) may drift on model upgrade (source: pr-review-toolkit:review-pr) — `docs/delegation.md:52`
- [ ] [doc] `docs/adr/README.md` ADR template missing `## Options Considered` section — AGENTS.md says ADRs document "options considered" but template has only Context/Decision/Consequences (source: pr-review-toolkit:review-pr) — `docs/adr/README.md`
- [ ] [doc] `docs/eval-criteria.md` Correctness criterion "How to test: Run the feature manually" — replace with specific pytest command or scenario list for CI-reproducibility (source: review) — `docs/eval-criteria.md:37`

## Review Backlog

### PR #97 — [REFACTOR] Move rebalance data assembly into service; guard USD conversion (2026-05-23)

- [ ] [debt] `RebalanceService()` instantiated per-request in `rebalance.py:_build_rebalance_plan` — inconsistent with container injection pattern; expose `container.rebalance_service` — `web/routes/rebalance.py:19`
- [ ] [debt] `portfolio_insight_service._build_rebalance_plan` duplicates `RebalanceService.build_plan_from_repos` — delegate to `self._rebalance_service.build_plan_from_repos(...)` — `services/portfolio_insight_service.py:294`

## From PR #69 (feat/per-account-rebalance) — 2026-04-24

- ~~**Optimize `_pick_next_group` sort key**~~ — Skipped. Sort key (`need_by_group[g]`) mutates every iteration; a priority queue would need decrease-key + stale-entry handling for a 5-element collection. O(5 log 5) per iteration is effectively free. (`rebalance_service.py:866-868`)

## Follow-up from feat/stock-service-extraction (PR #70)

- [x] **Consolidate duplicate name-resolution flows** — Added `StockService.persist_name(stock, raw_name)` shared helper; `portfolio_service.py` and `kis_account_sync_service.py` delegate to it when `stock_service` is injected. Creation path in `kis_account_sync_service.py:138` kept using `format_stock_name` directly (no Stock instance yet). (#71)

## Review Backlog

### PR #70 — [REFACTOR] Extract StockService (2026-04-25)

- [x] [debt] Redundant `StockService` initialization: replaced second construction with `set_price_service()` setter — single object identity across container lifecycle. (#71)
- [x] [debt] `_build_stock_name_map` no-price_service path: added `has_price_service` property to `StockService`. Fast-exit in routes left unchanged — existing test confirmed that stocks with already-persisted names should be returned even when `price_service` is unavailable. (#71)

## Review Backlog

### PR #71 — [REFACTOR] StockService API (2026-04-25)

- [x] [debt/blocked] Remove `else` fallback in `portfolio_service.py:149-152` — made `stock_service` a required keyword-only arg; updated all test call sites with `StockService(stock_repo)` or `Mock()`. (#77)
- [x] [doc/blocked] `persist_name` docstring: updated to single-path description. (#77)

## Review Backlog

### PR #77 — [REFACTOR] PortfolioService — require stock_service, drop else fallback (2026-04-28)

- [x] [debt] Constructor shape: move all optional collaborators in `PortfolioService.__init__` to keyword-only — currently optional positional args precede the required keyword-only `stock_service` (source: Claude) — `portfolio_service.py:76-87`
- [x] [debt] `_make_stock_service` helper vs inline inconsistency — centralize in `conftest.py` or always inline across test files (source: Claude) — `tests/services/test_portfolio_service.py:15-16`
- [x] [constraint] No test for `persist_name` raising — if `stock_repository.update_name` fails, `get_portfolio_summary` crashes silently; add error-path test or explicit except-and-log (source: Claude) — `portfolio_service.py:147`

## Review Backlog

### PR #81 — [FIX] Remove over-conservative cold-start guard for KIS key set 2 (2026-05-04)

- [x] [constraint] Add unit test for dual-key-set init path: assert `_build_kis_client_set` is called twice when `KIS_APP_KEY_2` is set, and that a second-call exception does not prevent key-set-1 init (source: Claude) — `container.py:308-323` (#83)

## Review Backlog

### PR #72 — [HARNESS] Clear backlog — log rotation, Q&A deadline, KST docs, tasks schema (2026-04-25)

- [x] [doc] `runbook.md` KST 섹션에 영향받는 테이블/모델명 명시 — 6개 모델 + `OrderExecutionModel` 예외 + `BaseModel.save()` 메커니즘 명시. (#75)
- [x] [debt] 멀티-세대 로그 백업 — `_LOG_BACKUP_COUNT = 5` 도입, `.log.1`~`.log.5` 유지, 다세대 테스트 추가. (#76)

## Review Backlog

### PR #85 — [FEAT] Add KisDomesticInvestorClient for daily investor flow (2026-05-04)

- [x] [debt] Add individual investor fields (`individual_net_qty`, `individual_net_krw`) to `DomesticInvestorFlow` — KIS `prsn_ntby_qty` / `prsn_ntby_tr_pbmn` now mapped; test added (PR TBD)

### PR #86 — [FEAT] Stage 2 — investor flow persistence layer + KisDomesticInvestorClient wiring (2026-05-05)

- [x] [debt] Consider `BigIntegerField` for KRW fields if DB backend changes — promoted to `BigIntegerField` for Postgres/MySQL portability (#88)
- [x] [debt] Narrow `IntegrityError` catch in `InvestorFlowRepository.save` — replaced with refetch-first pattern; same fix applied to `StockPriceRepository` (#88)
- [x] [debt] Align `DomesticInvestorFlow.date` (str) naming/type with repository's `flow_date` (date) — renamed field, added `_parse_yyyymmdd` helper (#88)

### PR #87 — [FEAT] Add restrict-overseas toggle to rebalancing (2026-05-06)

- [x] [debt] `_calculate_sell_amounts_by_account_group` computes sell targets for restricted overseas groups unnecessarily — added `restrict_overseas` + `positions` params; pre-builds eligible key set to skip overseas-only groups (#89)
- [x] [doc] Add comment to `is_domestic_ticker` documenting the 6-char heuristic assumption and known limitation (e.g., 6-char overseas tickers would be misclassified) — extended docstring (#89)
