<!--
Schema / lifecycle:
  Active sprint  — open [ ] items grouped by PR or feature branch below this header.
  Dormant        — no active sprint; file starts with "# (dormant)" and retains only open [debt/blocked] items.
-->

# Tasks — Deferred from PR reviews

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

### PR #72 — [HARNESS] Clear backlog — log rotation, Q&A deadline, KST docs, tasks schema (2026-04-25)

- [x] [doc] `runbook.md` KST 섹션에 영향받는 테이블/모델명 명시 — 6개 모델 + `OrderExecutionModel` 예외 + `BaseModel.save()` 메커니즘 명시. (#75)
- [x] [debt] 멀티-세대 로그 백업 — `_LOG_BACKUP_COUNT = 5` 도입, `.log.1`~`.log.5` 유지, 다세대 테스트 추가. (#76)
