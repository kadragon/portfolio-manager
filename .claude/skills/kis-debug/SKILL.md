---
name: kis-debug
description: >-
  Diagnose KIS Open Trading API failures in the portfolio-manager Go app —
  auth/token errors (EGW00123, 500), wrong-environment routing, overseas price
  returning empty/zero, account-sync errors (OPSQ2000 INVALID_CHECK_ACNO),
  multi-API-key sets (KIS_APP_KEY_2) not taking effect, and missing FX/exchange
  rates. Use this whenever a KIS sync button fails, prices don't render, a
  KIS-related error code or stack trace appears, exchange rates look wrong, or
  the user reports "동기화 실패", "가격 조회 안됨", "해외 주식", "환율", or pastes a
  koreainvestment.com / koreaexim.go.kr error — even if they don't name KIS
  explicitly. Maps symptom → root cause → check command or fix.
---

# KIS API Debug

Diagnostic decision-tree for KIS Open Trading API failures. Match the symptom, find the cause, run the check or apply the fix. Scoped to **debugging** — not a feature tutorial.

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

When a fix changes env or keys, the running server must be restarted to re-read `.env` and re-init the container.

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
Restart the server after fixing.

### 5. Multi-key set (`KIS_APP_KEY_2`) not taking effect

Key sets 2–9 come from `KIS_APP_KEY_{id}` / `KIS_APP_SECRET_{id}` (`container.go:454` `buildKISAuthExtra`); they inherit env/custType/baseURL/tokenManager from key-1. An account routes to its key set via `account.KisAPIKeyID` → `resolveSyncService` (`container.go:216`), which **falls back to key-1 if the id is unmapped**. So "key 2 not applying" is usually one of:

- Account row's `KisAPIKeyID` not set to 2 → silently uses key-1. Check the account's API-key setting in the UI / DB.
- `KIS_APP_KEY_2`/`KIS_APP_SECRET_2` missing or blank in `.env` → key set 2 never built → fallback to key-1.
- **Cold-start skip**: log line `KIS key set 2: skipping cold-start initialization to avoid 1-req/min rate-limit conflict with key set 1. Restart after ~60s to activate key set 2.` Key 2 only activates on a restart spaced **>60s** after key-1's token issuance. Fix: wait ~60s, restart server once.

```bash
grep -E '^KIS_APP_(KEY|SECRET)_[0-9]=' .env
docker compose logs web | grep -i 'key set'   # spot the cold-start-skip line
```

### 6. FX / exchange rate missing or wrong (overseas value shows ₩0)

`buildExchangeRate` priority (`container.go:481`): `USD_KRW_RATE` (fixed override) → `EXIM_AUTH_KEY` (live EXIM API) → **nil (no conversion → overseas values collapse to 0)**.

So overseas holdings showing ₩0 with correct quantity/price often means **no FX source configured**. EXIM client: `internal/services/exim_client.go` (base `https://oapi.koreaexim.go.kr`, `AP01`, returns `deal_bas_r`). The rate service caches with a 7-day backoff (`exchange_rate_service.go:32`), so a transient EXIM outage won't immediately drop the rate.

```bash
grep -E '^(USD_KRW_RATE|EXIM_AUTH_KEY)=' .env   # at least one must be set for FX
```

## Output contract

Always close a diagnosis with **(a) the named root cause** and **(b) a concrete next step** — a check command to confirm or the exact fix. Don't stop at "it's probably the env"; state which env value, what it should be, and the command to verify. If live verification is needed, use the `KIS_LIVE=1` test — never loop auth.
