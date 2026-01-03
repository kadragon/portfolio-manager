# AGENTS.md

## Development Notes
- Reference: https://github.com/koreainvestment/open-trading-api (KIS Open Trading API examples/specs) for ongoing implementation alignment.
- KIS token issuance is rate-limited (1 per minute); token cache stored at `.data/kis_token.json`.
- Local verification script: `scripts/check_kis_domestic_price.py` (loads `.env`, fetches token if needed, calls domestic price API).
- Local verification script: `scripts/check_kis_overseas_price.py` (loads `.env`, fetches token if needed, calls overseas price API; use `.venv/bin/python`).
- `KIS_ENV` accepts `real/prod` and `demo/vps/paper` (also tolerates `real/prod` form and whitespace).
- Overseas current price endpoint: `/uapi/overseas-price/v1/quotations/price` with TR ID `HHDFS00000300`.
