---
name: kis-debug
description: >-
  Diagnose KIS Open Trading API and Toss Invest API failures in the
  portfolio-manager Go app — KIS auth/token errors (EGW00123, 500),
  wrong-environment routing, overseas price returning empty/zero, account-sync
  errors (OPSQ2000 INVALID_CHECK_ACNO), multi-API-key sets (KIS_APP_KEY_2) not
  taking effect, missing FX/exchange rates; Toss client silently nil (missing
  TOSS_CLIENT_ID/SECRET), Toss OAuth/order errors, TossAccountSeq not linked.
  Use when a `pm sync`/`pm toss`/`toss-order-manage`/`rebalance-order` call
  fails, prices don't render, a KIS or Toss error code or stack trace appears,
  exchange rates look wrong, or the user reports "동기화 실패", "가격 조회 안됨",
  "해외 주식", "환율", "토스 주문 실패", "토스 동기화 안됨", or pastes a
  koreainvestment.com / koreaexim.go.kr / tossinvest.com error — even without
  naming KIS or Toss. Maps symptom → root cause → fix.
---

# KIS / Toss API Debug

Diagnostic decision-tree for KIS Open Trading API and Toss Invest API failures. Match the symptom, find the cause, run the check or apply the fix. Scoped to **debugging** — not a feature tutorial. KIS sections are below; Toss sections are under [Toss API failures](#toss-api-failures) further down.

## Hard safety rule — read first

**Token issuance is rate-limited to 1 request/minute.** Never loop live auth to "verify" a fix. One failed token call locks you out for ~60s. To check live behavior safely, use the guarded live test (below), which reuses the cached token — do **not** write throwaway scripts that hit `/oauth2/tokenP` in a loop.

```bash
# Safe live check — reuses cached .data/kis_token_*.json, no forced refresh.
KIS_LIVE=1 go test ./internal/kis/ -run TestLiveOverseasPriceRaw -v
```
Source: `internal/kis/overseas_price_live_test.go:24` (skips unless `KIS_LIVE=1`; CI never sets it).

## How the integration is wired

- Composition root: `internal/container/container.go` — `buildKISAuth` (primary key), `buildKISAuthExtra` (key sets 2–9), `buildExchangeRate` (FX).
- KIS clients: `internal/kis/` — auth, token manager, price (domestic/overseas/unified), balance/sync, error handlers, parsers.
- Token cache files: `.data/kis_token_{n}.json` (n = key-set id). RFC3339 expiry.
- Error format raised app-side: `KIS API error {msg_cd}: {msg1}` (`internal/kis/domestic_balance.go:146`).

Env/key changes take effect on the very next `pm` invocation — each call boots a fresh container, no restart needed.

## Symptom → cause → action

### 1. Token / auth errors (HTTP 500, `EGW00123`, "Server disconnected")

`EGW00123` = access token expired. The client already auto-refreshes and retries once (`internal/kis/error_handler.go:11` detects it; `base_client.go:52,81` `GetWithRetry`/`postWithRetry` refresh + retry). So a *single* EGW00123 in logs that then succeeds is normal.

Persistent 500 / "Server disconnected without sending a response":
- **Most likely rate-limit lockout** from repeated token issuance (1/min). Stop retrying, wait 60s, try once.
- Stale/corrupt token file → delete `.data/kis_token_{n}.json` and let it re-issue *once*.
- Bad credentials → check `KIS_APP_KEY`/`KIS_APP_SECRET` in `.env` are non-empty and match the env (real vs demo keys differ).

```bash
grep -E '^KIS_(APP_KEY|APP_SECRET|ENV|CUST_TYPE)=' .env   # confirm primary creds + env
ls -la .data/kis_token_*.json                              # token cache state
```

### 2. Wrong environment — silently hitting production

`KIS_ENV` is lowercased and truncated at the first `/` (`container.go:264-270`). Routing (`container.go:277-280`):
- `demo` / `vps` / `paper` → `https://openapivts.koreainvestment.com:29443` (paper trading)
- **anything else, including typos and empty → `https://openapi.koreainvestment.com:9443` (PRODUCTION)**

So `KIS_ENV=prod`, `KIS_ENV=production`, `KIS_ENV=relal` all silently route to live production. TR-IDs follow the same split (`internal/kis/base_client.go:25` `TrIDForEnv`: `real`/`prod` → real TR-ID, `demo`/`vps`/`paper` → demo TR-ID), so a real key with a demo TR-ID (or vice-versa) yields auth/permission errors.

Check the resolved env matches the intended account class (the app's only KIS-synced account is the ISA, which is a **real** account → `KIS_ENV=real`).

```bash
grep -E '^KIS_ENV=' .env
```

### 3. Overseas price empty or zero

Overseas quotation endpoints require the **3-letter** exchange code; orders use the 4-letter form — opposite conventions (`internal/kis/unified_price.go:170` `shortExchangeCode`): `NASD→NAS`, `NYSE→NYS`, `AMEX→AMS`. Missing this conversion → endpoint returns empty/zero, not an error.

- Confirm the call path runs through `shortExchangeCode` before the wire: `FetchCurrentPrice` (`overseas_price.go:41`), `FetchHistoricalClose` (`overseas_price.go:73`), `FetchBasicInfo` (`unified_price.go:129`).
- Symbol routing: ticker ≤4 chars → domestic client, else overseas (`unified_price.go:38`). A 6-digit domestic-looking ticker that is actually a US listing will be misrouted — the repo treats tickers **<6 digits** as overseas-listed (per the rebalancing rule). Mismatch here causes empty prices.
- A new exchange code not in the `NASD/NYSE/AMEX` switch passes through unchanged → likely empty. Add the mapping if a new venue appears.

```bash
KIS_LIVE=1 go test ./internal/kis/ -run TestLiveOverseasPriceRaw -v   # confirms live overseas fetch
```

### 4. Account sync fails — `OPSQ2000 ERROR : INPUT INVALID_CHECK_ACNO`

Account number mismatch on the balance endpoint. `KIS_CANO` (8-digit) + `KIS_ACNT_PRDT_CD` (2-digit product code) must match the actual KIS account. A full account string splits as `CANO(8) + PRDT(2)`:
- `6409798201` → `KIS_CANO=64097982`, `KIS_ACNT_PRDT_CD=01`

Common causes: pasting the full 10-digit string into `KIS_CANO`, swapping CANO/PRDT, or syncing a non-KIS account. **Only the ISA is a KIS account** — other accounts must not be synced. Sync uses `FetchAccountSnapshot` with TR-ID `TTTC8434R` (real) / `VTTC8434R` (demo) (`internal/kis/domestic_balance.go:26`).

```bash
grep -E '^KIS_(CANO|ACNT_PRDT_CD)=' .env   # CANO must be 8 digits, PRDT 2 digits
```
Each `pm` invocation is a fresh process, so a `.env` fix takes effect on the very next call — no
restart needed.

### 5. Multi-key set (`KIS_APP_KEY_2`) not taking effect

Key sets 2–9 come from `KIS_APP_KEY_{id}` / `KIS_APP_SECRET_{id}` (`container.go:454` `buildKISAuthExtra`); they inherit env/custType/baseURL/tokenManager from key-1. An account routes to its key set via `account.KisAPIKeyID` → `resolveSyncService` (`container.go:216`), which **falls back to key-1 if the id is unmapped**. So "key 2 not applying" is usually one of:

- Account row's `KisAPIKeyID` not set to 2 → silently uses key-1. `pm account list` doesn't expose
  this field at all (avoids a CodeQL clear-text-logging alert) — check it in the DB directly.
- `KIS_APP_KEY_2`/`KIS_APP_SECRET_2` missing or blank in `.env` → key set 2 never built → fallback to key-1.
- **Cold-start skip**: log line `KIS key set 2: skipping cold-start initialization to avoid 1-req/min rate-limit conflict with key set 1. Restart after ~60s to activate key set 2.` printed to stderr on
  `pm` startup. Since each `pm` call is a fresh process, key 2 only activates on a call made
  **>60s** after key-1's token issuance in an earlier call — fix: wait ~60s, run the `pm`/`sync`
  command again.

```bash
grep -E '^KIS_APP_(KEY|SECRET)_[0-9]=' .env
go run ./cmd/pm sync -account "<name>" 2>&1 | grep -i 'key set'   # spot the cold-start-skip line
```

### 6. FX / exchange rate missing or wrong (overseas value shows ₩0)

`buildExchangeRate` priority (`container.go:481`): `USD_KRW_RATE` (fixed override) → `EXIM_AUTH_KEY` (live EXIM API) → **nil (no conversion → overseas values collapse to 0)**.

So overseas holdings showing ₩0 with correct quantity/price often means **no FX source configured**. EXIM client: `internal/services/exim_client.go` (base `https://oapi.koreaexim.go.kr`, `AP01`, returns `deal_bas_r`). The rate service caches with a 7-day backoff (`exchange_rate_service.go:32`), so a transient EXIM outage won't immediately drop the rate.

```bash
grep -E '^(USD_KRW_RATE|EXIM_AUTH_KEY)=' .env   # at least one must be set for FX
```

### 7. One ticker permanently or intermittently missing after `price-sync`

Two related failure modes, both fixed as of 2026-07-11 — if you still see this, check `git log` for these fixes rather than re-diagnosing from scratch:

**(a) Permanent — legacy short-form exchange code in `stocks.exchange`.** Pre-migration data (and any manually-edited row) can hold the 3-letter price-endpoint form (`NAS`/`NYS`/`AMS`) instead of the canonical 4-letter order-form (`NASD`/`NYSE`/`AMEX`) that `prioritizedExchanges` expects. A mismatch silently falls back to the NASD-first default order for every fetch. If the ticker isn't NASDAQ-listed, that first attempt returns a success-but-empty response — and (before the fix below) the loop accepted it as final, so the price could never update. `prioritizedExchanges` (`internal/kis/unified_price.go:17`) now normalizes via `orderExchangeMap` before matching, so this self-heals on the next sync (`UpdateExchange` writes the canonical form back once a fetch finally succeeds). Confirm no stock still holds a short-form value:
```bash
sqlite3 .data/portfolio.db "select ticker, exchange from stocks where exchange in ('NAS','NYS','AMS');"
```

**(b) Transient — one empty response with no retry.** `UnifiedPriceClient.getOverseasPrice` used to `break` after the first non-error response even when it parsed to a zero/empty quote, never trying the other two exchanges. Fixed: it now only stops early once `quote.Price > 0` (`internal/kis/unified_price.go:112-118`, regression test `TestUnifiedPriceClientGetOverseasPriceFallsBackWhenPreferredExchangeEmpty`). Separately, `PriceSyncService.SyncOnce` had no retry at all for a failed/empty `GetPrice` call — one glitch meant that ticker was stuck on its prior price for the whole sync pass. Fixed: one bounded retry after `syncCallDelay` (`internal/services/price_sync_service.go:69-75`, regression test `TestPriceSyncServiceRetriesTransientFailure`).

Diagnosis for a *new* case of "ticker didn't update": confirm the ticker fetches fine right now and check whether today's row is actually missing/stale in the DB.

```bash
KIS_LIVE=1 go test ./internal/kis/ -run TestLiveOverseasPriceRaw -v   # add the ticker/exchange to the `cases` table if not QQQ/SPY
sqlite3 .data/portfolio.db "select ticker, price_date, price, exchange from stock_prices where ticker='<TICKER>' order by price_date desc limit 5;"
sqlite3 .data/portfolio.db "select ticker, exchange from stocks where ticker='<TICKER>';"   # short-form value here means (a) above, elsewhere
```

### 8. `1d` change rate shows exactly `0` for domestic (KRW) tickers, but not for overseas ones

Usually not a bug — a weekend/holiday sync artifact. `GetStockChangeRates` (`internal/services/price_service.go:107`) compares today's stored price to the price on `prevBizDay(today - 1)`. `SyncOnce` (`internal/services/price_sync_service.go:56,93`) saves every ticker — domestic and overseas alike — under the *same* KST calendar date, so this isn't a price_date/timezone discrepancy between markets. The actual difference is what the KIS quote endpoint returns on that call: when KRX is closed, the domestic endpoint returns the last traded (Friday) price completely unchanged, which lands in the DB as an exact duplicate of the prior trading day's row — and since `1d`'s lookback target resolves to that same prior day, current == past → rate is exactly `0`. Overseas tickers usually don't show this because the KIS overseas endpoint tends to keep returning a moving quote (the NYSE/NASDAQ session can still be active or freshly closed within the same KST calendar day) even when it's a KST weekend.

Identical back-to-back rows are **consistent with** this artifact but don't prove it on their own — two real trading sessions can coincidentally close at the same price. Treat an identical pair as corroborating evidence, and if in doubt confirm against the actual market calendar (was KRX/the relevant exchange open on that date) rather than treating equality alone as proof:
```bash
sqlite3 .data/portfolio.db "select ticker, price_date, price from stock_prices where ticker='<TICKER>' order by price_date desc limit 2;"
```
If the two rows are identical and the earlier date was a non-trading day for that ticker's market, the `0%` is expected; re-run `price-sync` on the next trading day to get a real `1d` value.

## Toss API failures

Toss has a much thinner error surface than KIS: no file-cached token, no env-name routing to
typo, and no code-to-action dispatch table in `internal/toss` (unlike KIS's `error_handler.go`).
Most Toss diagnosis is "read the wrapped error text" rather than "match a known code."

### 9. Toss client silently `nil` — every Toss call fails with "not configured"

`buildTossClient` (`internal/container/container.go:620-629`) reads `TOSS_CLIENT_ID` /
`TOSS_CLIENT_SECRET`; if either is blank, it returns `nil` **without raising an error** —
`Container.TossClient` stays nil and `TossAccountSync` is never built. The error only surfaces
later at the call site, not at startup:

- `pm toss <verb>`: `"toss client not configured (.env TOSS_CLIENT_ID/TOSS_CLIENT_SECRET)"` (`cmd/pm/toss.go:112-116`)
- `toss-order-manage`: `"Toss is not configured (.env TOSS_CLIENT_ID/TOSS_CLIENT_SECRET)"` (`cmd/toss-order-manage/main.go:121`)
- `rebalance-order` / order execution: `"account %q is Toss-linked but no Toss client is configured"` (`internal/services/order_execution_service.go:109`)

```bash
grep -E '^TOSS_(CLIENT_ID|CLIENT_SECRET)=' .env   # both must be non-empty
```

### 10. Account not linked to Toss

`"account %q is not linked to a Toss accountSeq"` (`cmd/pm/toss.go:126-128`,
`cmd/toss-order-manage/main.go:481-483`) means the account row has no `TossAccountSeq` set —
that's a portfolio-data `account update -toss-account-seq` task, not a Toss API problem.

### 11. Toss auth/OAuth errors

OAuth2 client-credentials flow, `internal/toss/client.go:179-221`. POSTs to
`{TOSS_BASE_URL}/oauth2/token`; the token is cached **in-memory only** for the process lifetime
(no `.data/*.json` file like KIS) and reused until ~1 minute before expiry (`client.go:184`).
Auth failure format: `"toss auth HTTP {status}: {error}: {error_description}"`
(`parseOAuthError`, `internal/toss/http.go:144-156`) — e.g. an `invalid_client` error means the
client ID/secret pair itself is wrong, not an expired token.

Token issuance is documented as **rate-limited to ~1/min** at the API level
(`internal/toss/client_live_test.go:58`), same caution as KIS: don't loop live Toss auth calls
to "verify" a fix.

### 12. Toss API errors (orders, market data, account)

Non-auth failures go through `parseAPIError` (`internal/toss/http.go:132-142`):
`"{prefix} HTTP {status}: {code}: {message} (request_id={id})"` — `{prefix}` is a call-site
string like `"toss conditional order"` or `"toss price-limits"`, so the prefix itself tells you
which endpoint failed. Read `{code}`/`{message}` as the actual signal; there's no app-side
code-to-action table to consult, unlike KIS's `EGW00123`. Codes seen in this repo's tests only
(not a documented exhaustive list) include `INVALID_ACCOUNT`, `RATE_LIMIT_EXCEEDED` — treat
`RATE_LIMIT_EXCEEDED` as "back off and retry later," not as a config problem.

There's no client-side pre-check for market-closed or insufficient-balance conditions (e.g.
`create-amount` outside US regular hours) — those come back as a Toss server error through
`parseAPIError`, not a distinct app error path. Confirm market hours with
`pm toss market-calendar-us` before assuming the error text is wrong.

### 13. Wrong Toss base URL

Unlike `KIS_ENV`, Toss has **no sandbox/live split to route between** — there is one
`defaultBaseURL = "https://openapi.tossinvest.com"` (`internal/toss/client.go:23`), only
overridable via `TOSS_BASE_URL` (`container.go:626`). So a "wrong environment" symptom here
almost always means `TOSS_BASE_URL` is explicitly (and probably accidentally) set to something
else, not a typo silently falling back to production the way `KIS_ENV` does:

```bash
grep -E '^TOSS_BASE_URL=' .env   # unset is normal; if set, confirm it's intentional
```
`toss-order-manage` prints the resolved base URL as a live-money warning on every run
(`"Toss base URL=%q — this places a LIVE order-management action against real money"`,
`main.go:123`) — treat that banner the same way as the `KIS_ENV` banner in
execute-rebalance-plan: read it before confirming any `-yes` run.

### 14. Live Toss check

```bash
TOSS_LIVE=1 go test ./internal/toss -run TestLiveFetchAccountSnapshot -count=1 -v
```
Documented in `docs/runbook.md` §Live API smoke tests, gated by `internal/toss/client_live_test.go:18-19`. Needs
`TOSS_CLIENT_ID`/`TOSS_CLIENT_SECRET` (optional `TOSS_ACCOUNT_SEQ`/`TOSS_BASE_URL`). Read-only —
`TestLiveTossReadEndpoints` sweeps read endpoints only; live tests never place a real order
(`client_live_test.go:61-63`). Same rule as KIS: don't write a throwaway script that hits
`/oauth2/token` in a loop.

## Output contract

Always close a diagnosis with **(a) the named root cause** and **(b) a concrete next step** — a check command to confirm or the exact fix. Don't stop at "it's probably the env"; state which env value, what it should be, and the command to verify. If live verification is needed, use the `KIS_LIVE=1` (KIS) or `TOSS_LIVE=1` (Toss) test — never loop auth.
