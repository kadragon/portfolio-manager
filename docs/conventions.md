# Conventions

What agents frequently get wrong in this codebase. Ruff/pyright catch style and types — this doc covers patterns the tools miss.

## Naming

| Element | Pattern | Example |
|---------|---------|---------|
| Python files | `snake_case` | `account_repository.py` |
| Classes | `PascalCase` | `AccountRepository`, `KisDomesticPriceClient` |
| Functions / methods | `snake_case` | `get_by_id()`, `sync_account()` |
| Peewee ORM model classes | `{Name}Model`, singular | `AccountModel`, `HoldingModel`, `StockPriceModel` |
| Domain dataclasses | `PascalCase`, singular | `Account`, `Holding`, `StockPrice` |
| Repository classes | `{Model}Repository` | `AccountRepository`, `HoldingRepository` |
| KIS client classes | `Kis{Scope}{Role}Client` | `KisDomesticPriceClient`, `KisOverseasOrderClient` |
| Route files | `snake_case`, plural noun | `accounts.py`, `groups.py` |
| Template dirs | match route file name | `templates/accounts/`, `templates/groups/` |
| HTMX partial templates | `_` prefix | `_form.html`, `_row.html` |
| Environment vars | `SCREAMING_SNAKE_CASE` | `KIS_APP_KEY`, `USD_KRW_RATE` |
| Commit messages | `[TYPE] description` | `[FEAT] add account sync endpoint` |

## Commit Types

`[FEAT]` · `[FIX]` · `[REFACTOR]` · `[DOCS]` · `[CONSTRAINT]` · `[HARNESS]` · `[PLAN]`

- `[FEAT]` — new user-visible behavior
- `[FIX]` — bug fix, must include a test that would have caught it
- `[REFACTOR]` — structural change only, no behavior change
- `[CONSTRAINT]` — new structural test / lint rule
- `[HARNESS]` — CI, sweep, tooling changes
- `[PLAN]` — backlog / tasks updates only

## HTMX Route Pattern

Routes must check the `HX-Request` header and return the appropriate fragment or full page:

```python
# correct — consistent partial/full pattern
@router.get("/accounts/{id}/edit")
async def edit_account(request: Request, id: int, container=Depends(get_container)):
    account = container.account_repository.get_by_id(id)
    if request.headers.get("HX-Request"):
        return templates.TemplateResponse("accounts/_form.html", {"request": request, "account": account})
    return templates.TemplateResponse("accounts/edit.html", {"request": request, "account": account})
```

Never return a full layout template in response to an HTMX partial request.

## Peewee Model Usage

- Models live in `models/`. They define schema (fields, meta, indexes) — no business logic.
- All queries go through repositories. Never call `Model.select()` or `Model.get()` directly outside `repositories/`.
- Use `database_proxy` from `services/database.py` for all model `Meta.database` assignments so tests can swap to in-memory SQLite.

```python
# correct
class Account(Model):
    class Meta:
        database = database_proxy

# violation — hardcoded DB handle
class Account(Model):
    class Meta:
        database = SqliteDatabase("portfolio.db")
```

## KIS Client Usage

- Never call `manager.get_token()` manually in route handlers or services — clients do this internally.
- Token issuance is rate-limited to **1 per minute** (KIS API constraint). If a test triggers token issuance, mark it `@pytest.mark.integration`.
- `KIS_ENV` accepts: `real`, `demo`, `vps`, `paper`. Anything else falls through to the production URL silently — validate at startup.
- Use `KisUnifiedPriceClient` / `KisUnifiedOrderClient` for market-agnostic operations; use the domestic/overseas clients directly only when you need market-specific behavior.

## Testing

- Unit tests: no external calls, no filesystem side-effects, no sleep.
- Integration tests: `@pytest.mark.integration` — CI excludes these. Run locally with `uv run pytest -m integration`.
- Test fixtures in `tests/conftest.py`: use `test_container` (in-memory SQLite) — never instantiate `ServiceContainer` directly in tests.
- Coverage threshold is 85% (branch coverage). CI fails below this. New features need tests before merge.
- Do NOT mock repository methods in unit tests for routes — use the in-memory DB via the test container fixture.

## Error Handling

- Services raise `ValueError` or domain-specific exceptions for expected failures (invalid input, not found).
- Routes catch domain exceptions and return appropriate HTTP status codes via `HTTPException`.
- KIS API errors surface via `KisApiError` (in `services/kis/kis_api_error.py`) — don't wrap in generic `Exception`.
- Never swallow exceptions silently. Log at `WARNING` or above, then re-raise or convert.

## Environment Variables

All credentials and configuration go through `.env` + `os.getenv()`. No hardcoded values in source. Required vars are documented in `docs/runbook.md`. Optional vars have sensible defaults in code.
