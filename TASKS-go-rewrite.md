# Remaining Tasks — Python → Go rewrite

Branch `feat/go-rewrite`. Companion: `handoff-go-rewrite.md` (conventions, parity
facts, gotchas), plan `~/.claude/plans/witty-wobbling-lemur.md`.

Done: **Phase 0** (scaffold + DB types) · **Phase 1** (groups slice).
Each unchecked box = one route/repo-method/template/service to port. Per-slice
exit criterion: `make go-build go-vet go-lint go-test` green + arch test green +
`scripts/parity_check.sh` routes MATCH the Python oracle.

---

## Phase 2 — Stocks slice

Python: `web/routes/groups.py` (stock routes), `repositories/stock_repository.py`,
`services/stock_name_formatter.py`, templates `groups/stocks.html`,
`groups/_stock_row.html`, `groups/_stock_form.html`.

Routes:
- [ ] `GET  /groups/{group_id}/stocks` — full page (group + stocks list)
- [ ] `POST /groups/{group_id}/stocks` — create (ticker upper-cased) → `_stock_row`
- [ ] `GET  /groups/{group_id}/stocks/{stock_id}` — row partial (cancel)
- [ ] `GET  /groups/{group_id}/stocks/{stock_id}/edit` — edit form (+ groups dropdown)
- [ ] `PUT  /groups/{group_id}/stocks/{stock_id}` — update ticker + optional move-group; 422 empty ticker, 404 bad target group, 422 bad UUID
- [ ] `DELETE /groups/{group_id}/stocks/{stock_id}` — 200 empty

Repository `StockRepository` (model `stocks`):
- [ ] `create(ticker, group_id, name="")`
- [ ] `list_by_group(group_id)`
- [ ] `list_all()`
- [ ] `get_by_id(id)` → nil if absent
- [ ] `get_by_ticker(ticker)` → nil if absent
- [ ] `update(id, ticker)`
- [ ] `update_group(id, group_id)`
- [ ] `update_exchange(id, exchange)`
- [ ] `update_name(id, name)`
- [ ] `delete(id)`

Other:
- [ ] `stock_name_formatter` — strip ETF suffix `증권상장지수투자신탁(주식)`
- [ ] domain `models.Stock` (id, ticker, name, group_id, exchange nullable, timestamps)
- [ ] templ stocks page + `_stock_row` + `_stock_form`
- [ ] register routes; extend parity_check.sh

---

## Phase 3 — Accounts slice

Python: `web/routes/accounts.py`, `repositories/account_repository.py`, templates
`accounts/list.html`, `accounts/_row.html`, `accounts/_form.html`. (KIS sync button
deferred to Phase 8.)

Routes:
- [ ] `GET  /accounts` — full page
- [ ] `GET  /accounts/{id}` — row partial
- [ ] `GET  /accounts/{id}/edit` — edit form (KIS fields; api-key dropdown if >1 key)
- [ ] `POST /accounts` — create (name, cash_balance Decimal)
- [ ] `PUT  /accounts/{id}` — update name/cash/kis_account_no/kis_api_key_id; 422 + `HX-Retarget`/`HX-Reswap` on validation error; calls `validate_account` (defer KIS to P8 — stub/skip validation path or guard)
- [ ] `PUT  /accounts/bulk-cash` — dynamic `cash_{id}` fields; 422 or `HX-Refresh`
- [ ] `DELETE /accounts/{id}` — `delete_with_holdings`

Repository `AccountRepository` (model `accounts`):
- [ ] `create(name, cash_balance)`
- [ ] `list_all()`
- [ ] `update(id, name, cash_balance, kis_account_no?, kis_api_key_id?)` — `_UNSET` sentinel → Go pointer/option (distinguish "unchanged" from "set null")
- [ ] `delete_with_holdings(id)` — delete holdings then account (Go tx)

Other:
- [ ] `models.Account` (cash_balance numeric.Decimal, kis fields nullable)
- [ ] markupsafe.escape → templ auto-escape (account name in error msg)
- [ ] templ list + `_row` + `_form`; register; parity

---

## Phase 4 — Holdings slice

Python: `web/routes/accounts.py` (holdings routes),
`repositories/holding_repository.py`, templates `accounts/holdings.html`,
`accounts/_holding_row.html`, `accounts/_holding_form.html`,
`accounts/_holdings_rows.html`, `accounts/_holdings_bulk_result.html`.

Routes:
- [ ] `GET  /accounts/{id}/holdings` — full page (holdings, all_stocks, all_groups, stock_map, stock_name_map)
- [ ] `GET  /accounts/{id}/holdings/{hid}` — row partial
- [ ] `GET  /accounts/{id}/holdings/{hid}/edit` — edit form
- [ ] `POST /accounts/{id}/holdings` — create (stock_id, quantity) → `_holdings_rows`
- [ ] `PUT  /accounts/{id}/holdings/{hid}` — update quantity → `_holding_row`
- [ ] `POST /accounts/{id}/holdings/by-ticker` — ticker + qty + group_id/new_group_name; error → `HX-Retarget`/`HX-Reswap` to `#by-ticker-error` 422
- [ ] `PUT  /accounts/{id}/holdings/bulk` — `holding_id[]`+`quantity[]`; `_holdings_bulk_result` w/ `hx-swap-oob`
- [ ] `DELETE /accounts/{id}/holdings/{hid}` — 200 empty

Repository `HoldingRepository` (model `holdings`):
- [ ] `create(account_id, stock_id, quantity)`
- [ ] `list_by_account(account_id)`
- [ ] `update(id, quantity)`
- [ ] `delete(id)`
- [ ] `delete_by_account(account_id)`
- [ ] `bulk_update_by_account(account_id, [(id,qty)])` — Go tx; validate all belong to account, no dups, qty>0
- [ ] `get_aggregated_holdings_by_stock()` — `SELECT stock_id, SUM(quantity) GROUP BY stock_id`

Other:
- [ ] `models.Holding`; 5 templ partials; OOB swap parity; register; parity

---

## Phase 5 — Deposits slice

Python: `web/routes/deposits.py`, `repositories/deposit_repository.py`, templates
`deposits/list.html`, `deposits/_row.html`, `deposits/_form.html`.

Routes:
- [ ] `GET  /deposits` — full page (deposits, total)
- [ ] `GET  /deposits/{id}` — row partial
- [ ] `GET  /deposits/{id}/edit` — edit form
- [ ] `POST /deposits` — create (amount, deposit_date, note?); duplicate date → update + `HX-Refresh`
- [ ] `PUT  /deposits/{id}` — update
- [ ] `DELETE /deposits/{id}` — 200 empty

Repository `DepositRepository` (model `deposits`):
- [ ] `create(amount, deposit_date, note?)`
- [ ] `update(id, amount?, deposit_date?, note?)` — `_UnsetNote` sentinel for note
- [ ] `list_all()` — `ORDER BY deposit_date DESC`
- [ ] `get_by_date(deposit_date)` → nil
- [ ] `delete(id)`
- [ ] `get_total()` — `SUM(amount)`
- [ ] `get_first_deposit_date()` — `ORDER BY deposit_date ASC LIMIT 1`

Other:
- [ ] `models.Deposit`; `format_krw` filter (₩{:,.0f}); 3 templ; register; parity

---

## Phase 6 — Portfolio + Price + Exchange + Dashboard

Largest slice. Brings in KIS price clients + decimal-heavy math.

Services:
- [ ] `PriceService` — in-mem + DB cache (`StockPriceRepository`), `get_stock_price`, `get_stock_change_rates` (1d/1m/6m/1y, business-day shift)
- [ ] `StockPriceRepository` — `get_by_ticker_and_date`, `save` (upsert, preserve name if new empty)
- [ ] `ExchangeRateService` — fixed `USD_KRW_RATE` or EXIM client; in-mem cache; 7-day backoff
- [ ] `EximExchangeRateClient` — GET exchangeJSON, parse USD deal_bas_r
- [ ] `PortfolioService` — `get_holdings_by_group`, `get_portfolio_summary` (KRW valuations, cash, invested, return rates)
- [ ] `compute_group_summary` — per-group actual/target/diff rows, sorted
- [ ] `stock_service` resolve/persist name (late-bound price_service)

KIS price stack:
- [ ] `kis_auth_client` (OAuth tokenP), `kis_token_manager` (1-min skew), `kis_token_store` (Memory + File `.data/kis_token_{key}.json`, KST-naive inference)
- [ ] `kis_base_client` (headers, retry on EGW00123, TR-id env routing)
- [ ] `kis_error_handler` (is_token_expired), `kis_api_error`
- [ ] `kis_market_detector` (is_domestic = 6-digit)
- [ ] `kis_price_parser` (parse_korea_price / parse_us_price → PriceQuote)
- [ ] `kis_domestic_price_client` (FHKST01010100 current, FHKST03010100 hist)
- [ ] `kis_overseas_price_client` (HHDFS00000300 current, HHDFS76240000 hist)
- [ ] `kis_domestic_info_client` / `kis_overseas_info_client` (name lookup)
- [ ] `kis_unified_price_client` (auto-route domestic/overseas)

Filters (templ funcs): `format_krw`, `format_usd`, `format_percent` (quantize 0.1 HALF_UP), `format_signed_percent`, `abs`, `format_rebalance_quantity` (KRW floor int / USD 6dp).

Routes/UI:
- [ ] `GET /` — dashboard (`dashboard.html`); replace placeholder
- [ ] templ dashboard; register; parity (decimal/quantize/floor exact)

`ServiceContainer` env wiring: KIS_ENV routing, app keys 1/2, EXIM, fixed rate.

---

## Phase 7 — Rebalance

Python: `web/routes/rebalance.py`, `services/rebalance_service.py`,
`services/rebalance_execution_service.py`, `repositories/order_execution_repository.py`,
templates `rebalance/view.html`, `rebalance/_result.html`.

- [ ] `RebalanceService.build_plan` — hardcoded group names (국내성장/국내배당/해외성장/해외안정/해외배당) + `_GROUP_BANDS`, region diagnostic; `build_plan_from_repos`
- [ ] `RebalanceExecutionService` — `create_order_intents`, `execute_rebalance_orders` (dry-run/live), qty floor int, overseas default NASD, KRW/USD by `is_domestic_ticker`
- [ ] `OrderExecutionRepository` — `create` (raw_response JSON), `list_recent(limit)`
- [ ] KIS order clients: `kis_domestic_order_client` (TTTC0012U/0011U), `kis_overseas_order_client` (TTTT1002U/1006U), `kis_unified_order_client` (route + fetch price)
- [ ] `GET /rebalance` (restrict_overseas query), `POST /rebalance/execute` (confirm, restrict_overseas)
- [ ] templ view + `_result`; register; parity (KIS live = Integration, skip CI)

---

## Phase 8 — KIS account sync

Python: `services/kis_account_sync_service.py`,
`services/kis/kis_domestic_balance_client.py`, accounts sync route + `_sync_result`.

- [ ] `KisDomesticBalanceClient.fetch_account_snapshot` — paginated `tr_cont` M/F + `ctx_area_fk100/nk100`, aggregate qty per ticker (TTTC8434R/VTTC8434R)
- [ ] `KisAccountSyncService.sync_account` — snapshot diff create/update/delete, `KisEmptySnapshotError` guard, sync-group fallback ("KIS 자동동기화"), `validate_account`
- [ ] JSONL rotating event log (10MB, 5 backups) → `.data/kis_sync.log`
- [ ] `POST /accounts/{id}/sync` (confirm_empty) → `_sync_result`; sync button in `accounts/_row`
- [ ] mark live tests `Integration` (GP-2); register; parity (mock balance client)

---

## Phase 9 — Insights / LLM

Python: `web/routes/insights.py`, `services/portfolio_insight_service.py`,
`services/llm/ollama_client.py`, `services/llm/prompt_templates.py`, templates
`insights/view.html`, `_narrative.html`, `_rebalance_xai.html`, `_qa.html`.

- [ ] `OllamaClient.chat` — POST /api/chat, tools, format json, `OllamaUnavailableError`
- [ ] prompt templates (Korean): narrative / rebalance-xai / qa / qa-json-fallback + 4 tool schemas
- [ ] `PortfolioInsightService` — `generate_narrative(daily/weekly)`, `explain_rebalance`, `answer_question` (4 tools: group_summary/top_movers/holding_value/deposit_history, 3-iter cap, JSON fallback)
- [ ] `GET /insights`, `GET /insights/narrative?period=`, `GET /insights/rebalance-xai`, `POST /insights/qa`
- [ ] templ view + 3 partials; register; parity (mock Ollama)
- [ ] env: OLLAMA_MODEL/HOST/TIMEOUT_SEC/NUM_CTX

---

## Phase 10 — Cutover

- [ ] full `ServiceContainer` env wiring complete + `close()` lifecycle
- [ ] move `src/.../web/static` + tailwind input → `internal/web/static`; update Makefile css paths + `staticDir()`
- [ ] Dockerfile → multi-stage Go (`CGO_ENABLED=0`, scratch/alpine); docker-compose → Go binary
- [ ] pre-commit → golangci-lint + `go test` + `templ generate --check` + `sqlc diff` (drop ruff/pyright/bandit/pytest)
- [ ] CI (`.github/workflows/ci.yml`) → Go-only; coverage gate 85% (exclude generated: `internal/db/sqlc`, `*_templ.go`, `cmd`)
- [ ] rewrite `docs/*.md` + `AGENTS.md` golden principles for Go toolchain
- [ ] `.claude/settings.local.json` permissions: `uv run`/pytest → go/golangci
- [ ] DELETE `src/`, `tests/` (py), `pyproject.toml`, `uv.lock`, `.venv`, `.bandit`, `.pre-commit-config.yaml` (py hooks)
- [ ] final 38-route parity sweep + `docker compose up` smoke
- [ ] remove `scripts/parity_check.sh` + `handoff`/`TASKS` md (oracle gone)
