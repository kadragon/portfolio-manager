# Architecture

FastAPI + HTMX 기반 포트폴리오 관리 앱. KIS Open Trading API로 국내/해외 주식을 연동한다.

## Stack

| Layer | Technology |
|-------|------------|
| Language | Python 3.13+ |
| Web framework | FastAPI + Jinja2 + HTMX |
| Frontend | Tailwind CSS v4 (standalone CLI), DaisyUI v5 |
| Database | SQLite via Peewee ORM |
| KIS API | 한국투자증권 Open Trading API (domestic + overseas) |
| Exchange rate | 한국수출입은행 EXIM API |
| Package manager | uv |
| CI | GitHub Actions |

## Source Layout

```
src/portfolio_manager/
  core/
    container.py          # ServiceContainer — DI root, wires all dependencies
  models/                 # Domain dataclasses (pure Python, no DB dependency)
  repositories/           # All DB access: *Repository classes (one per model group)
  services/
    kis/                  # KIS API clients (auth, token, price, order, balance)
    exchange/             # EXIM exchange rate client + service
    database.py           # Peewee ORM model definitions (*Model classes) + init_db / close_db
    portfolio_service.py  # Core portfolio logic
    price_service.py      # Price fetching + daily cache
    rebalance_service.py  # Rebalancing recommendation logic
    rebalance_execution_service.py
    kis_account_sync_service.py
  web/
    app.py                # FastAPI app factory + Jinja2 filters
    routes/               # Route handlers (accounts, dashboard, deposits, groups, rebalance)
    templates/            # Jinja2 + HTMX templates
    static/               # Built CSS + JS + favicon
    tailwind/             # Tailwind input CSS + DaisyUI bundle
  cli/                    # Rich terminal CLI (separate entry point)
tests/
  conftest.py             # Fixtures: in-memory SQLite DB, test container
  repositories/           # Repository unit tests
  services/               # Service unit tests (including KIS mocks)
  web/                    # Route + template tests
  migrations/             # Migration smoke tests
```

## Layer Rules

### Dependency Direction

```
web/ → services/ → repositories/ → services/database (Peewee ORM models)
                                 ↘ models/ (domain dataclasses)
```

Upper layers import lower layers. **Reverse imports are architectural violations.**

- `web/routes/` calls service methods via `ServiceContainer`; never imports repository classes or Peewee models directly.
- `services/` (business logic) uses repositories injected via constructor; never imports from `web/`.
- `repositories/` queries Peewee ORM models from `services/database`; never imports from other `services/` modules or `web/`.
- `models/` — pure Python dataclasses; no DB imports, no imports from other layers.
- `services/database.py` — Peewee `*Model` classes + `init_db`/`close_db`; no business logic imports.

### ServiceContainer is the DI root

All wiring happens in `core/container.py`. Route handlers receive the container (or specific services) via FastAPI dependency injection (`Depends`). Nothing outside `container.py` constructs repositories or clients directly.

### KIS client hierarchy

```
KisAuthClient + FileTokenStore → TokenManager
TokenManager → Kis*Client (base_client.py)
  ├── KisDomesticPriceClient
  ├── KisOverseasPriceClient
  ├── KisDomesticBalanceClient
  ├── KisDomesticOrderClient
  └── KisOverseasOrderClient
KisUnifiedPriceClient / KisUnifiedOrderClient  # facade over domestic + overseas
```

Each client shares the same `httpx.Client` and `TokenManager` per key set. `ServiceContainer` builds key sets keyed by integer ID (1, 2, …).

## Data Access

Two model layers exist:
- **`models/`** — Domain dataclasses (`Account`, `Group`, `Holding`, …). Returned by repositories, used in services and web.
- **`services/database.py`** — Peewee ORM models (`AccountModel`, `GroupModel`, …). Used only inside `repositories/`.

All DB queries go through `*Repository` classes in `repositories/`. Pattern:

```python
# correct — repository returns domain dataclass
account = container.account_repository.get_by_id(id)

# violation — direct Peewee model query outside repository
account = AccountModel.get_by_id(id)  # caught by tests/arch/test_layer_boundaries.py
```

DB initialization (`init_db()`) and teardown (`close_db()`) are in `services/database.py`. Peewee uses a proxy database so the connection can be swapped in tests (in-memory SQLite).

## Key Abstractions

1. **ServiceContainer** — single object that owns every live dependency. Route handlers should only interact with services/repositories exposed by the container.
2. **TokenManager** — handles KIS token refresh transparently. Clients call `manager.get_token()` before each request; the manager re-issues if expired (rate-limited to 1/min — see AGENTS.md).
3. **KisClientSet** — groups the three order/balance clients that share one app key pair. The container maps `key_id → KisClientSet`.
4. **Repository pattern** — separates query logic from business logic. Each repository exposes named methods (`list_accounts()`, `get_by_id()`, etc.) — never raw ORM queries at the call site.
5. **HTMX partial rendering** — routes check `HX-Request` header and return either a full page or a partial template fragment. Keep this pattern consistent across all route handlers.
