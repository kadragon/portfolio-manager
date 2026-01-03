# AGENTS.md

## Development Notes
- Reference: https://github.com/koreainvestment/open-trading-api (KIS Open Trading API examples/specs) for ongoing implementation alignment.
- KIS token issuance is rate-limited (1 per minute); token cache stored at `.data/kis_token.json`.
- Local verification script: `scripts/check_kis_domestic_price.py` (loads `.env`, fetches token if needed, calls domestic price API).
- Local verification script: `scripts/check_kis_overseas_price.py` (loads `.env`, fetches token if needed, calls overseas price API; use `.venv/bin/python`).
- `KIS_ENV` accepts `real/prod` and `demo/vps/paper` (also tolerates `real/prod` form and whitespace).
- Overseas current price endpoint: `/uapi/overseas-price/v1/quotations/price` with TR ID `HHDFS00000300`.

## Supabase Integration
- Supabase credentials stored in `.env`: `SUPABASE_URL` and `SUPABASE_KEY`.
- Database schema:
  - `groups` table: stores stock groups (id, name, created_at, updated_at)
  - `stocks` table: stores stock tickers (id, ticker, group_id, created_at, updated_at)
  - Relationship: groups 1:N stocks
- Migration file: `supabase/migrations/20260103000000_create_groups_and_stocks.sql`
- Repositories implemented:
  - `GroupRepository`: create(), list_all()
  - `StockRepository`: create(), list_by_group()
- Supabase client factory: `src/portfolio_manager/services/supabase_client.py`
- Data models: `src/portfolio_manager/models/group.py`, `src/portfolio_manager/models/stock.py`
- Test coverage: `tests/test_group_repository.py`, `tests/test_stock_repository.py`

## Strategic Insights
- TUI flows now split into `GroupListScreen` and `StockListScreen`; screens load data on mount and handle add/delete actions via Supabase repositories.
- Group selection in the TUI drives stock list context by passing the selected group ID into the stock screen.

## Governance Updates
- Authentication clients now share the `AuthClient` interface to decouple token management from a concrete provider.
- TUI, repository, and KIS client behaviors are covered by dedicated tests to enforce UI flows and API parsing expectations.
