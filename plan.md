# plan.md

- [x] Map overseas exchange codes (NAS/NYS/AMS) to order codes (NASD/NYSE/AMEX) in unified order client
- [x] Deduplicate KIS token-refresh retry logic into a shared helper on KisBaseClient
- [x] Consolidate price parsing so kis_price_parser.py is used by live clients
- [x] Normalize exchange codes across price/order flows by persisting canonical codes in stocks.exchange
