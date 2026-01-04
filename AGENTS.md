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
  - `groups` table: stores stock groups (id, name, target_percentage, created_at, updated_at)
  - `stocks` table: stores stock tickers (id, ticker, group_id, created_at, updated_at)
  - Relationship: groups 1:N stocks
- `accounts` table: stores brokerage accounts (id, name, cash_balance, created_at, updated_at)
- `holdings` table: stores account holdings (id, account_id, stock_id, quantity, created_at, updated_at)
- Relationship: accounts 1:N holdings, stocks 1:N holdings
- Migration files:
  - `supabase/migrations/20260103000000_create_groups_and_stocks.sql`
  - `supabase/migrations/20260103010000_create_accounts_and_holdings.sql`
  - `supabase/migrations/20260104000000_add_target_percentage_to_groups.sql`
- Repositories implemented:
  - `GroupRepository`: create(), list_all(), update(), delete()
  - `StockRepository`: create(), list_by_group()
  - `AccountRepository`: create(), list_all(), delete_with_holdings()
  - `HoldingRepository`: create(), list_by_account(), delete_by_account(), get_aggregated_holdings_by_stock()
- Services implemented:
  - `PortfolioService`: get_holdings_by_group() - aggregates holdings across accounts by stock and groups them by group
  - `PortfolioService`: get_portfolio_summary() - aggregates holdings with real-time price and valuation
  - `PriceService`: get_stock_price(ticker) - fetches current stock price from price client
  - `KisUnifiedPriceClient`: get_price(ticker) - routes to domestic/overseas KIS API based on ticker format
- Supabase client factory: `src/portfolio_manager/services/supabase_client.py`
- Data models: `src/portfolio_manager/models/group.py` (target_percentage 포함), `src/portfolio_manager/models/stock.py`, `src/portfolio_manager/models/account.py`, `src/portfolio_manager/models/holding.py`
- Test coverage: `tests/test_group_repository.py`, `tests/test_stock_repository.py`, `tests/test_account_repository.py`, `tests/test_account_delete_cascade.py`, `tests/test_holding_repository.py`, `tests/test_holding_quantity_decimal.py`, `tests/test_holding_aggregation.py`, `tests/test_portfolio_service.py`, `tests/test_rich_dashboard.py`

## Strategic Insights
- Rich-only CLI replaces Textual screens; menu navigation and prompts drive group/stock flows.
- Group selection now leads to a stock menu loop with table-based rendering and back navigation.
- List selections for groups/stocks/accounts/holdings use prompt_toolkit choice (arrow-key) instead of numeric input.
- Main menu displays unified portfolio dashboard in single table with price and valuation information.
- Dashboard shows: Group, Ticker, Quantity, Price, Value (USD는 KRW 환산), and Total Portfolio Value.
- Currency symbols display based on stock market: ₩ for KRW (domestic), $ for USD (overseas).
- 해외 종목명 누락 시 티커를 Name 컬럼에 표시하고, KIS 응답의 다양한 이름 필드로 보완한다.
- 그룹 목록에 목표 비중(%)을 함께 표시하고 추가/수정 시 입력을 받는다.

## Governance Updates
- Authentication clients now share the `AuthClient` interface to decouple token management from a concrete provider.
- Rich CLI flows and account/holding repositories are test-backed to lock in prompt/flow behavior and data parsing.
- Choice-based selection helpers cover group/account/stock/holding lists to standardize CLI selection inputs.
- Portfolio aggregation logic is encapsulated in PortfolioService with comprehensive test coverage for cross-account holding summation.
- Real-time pricing integrated via KIS API with automatic market detection (6-character codes = domestic, including alphanumeric like "0052D0"; alphabetic = overseas).
- Dashboard gracefully degrades to quantity-only display if KIS credentials unavailable or price fetch fails.
- Market detection uses length-based logic: 6-character tickers route to domestic API, others to overseas API (supports both pure numeric and alphanumeric Korean stock codes).
- Currency information flows from PriceQuote through PriceService and StockHoldingWithPrice to dashboard rendering with appropriate symbols (₩/$ based on KRW/USD).
- 해외 가격 조회는 거래소(NAS/NYS/AMS)를 순차 조회하며, 비어있는 종목명은 대체 필드로 채운다.
- USD 보유분은 `value_krw`에 환산 평가액을 저장해 대시보드에서 KRW 기준으로 출력한다.
- USD/KRW 환율 조회는 EXIM에서 USD 누락 시 최근 7일 내 직전 영업일로 자동 재시도한다.
