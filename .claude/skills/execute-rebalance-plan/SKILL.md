---
name: execute-rebalance-plan
description: Place real ISA/TOSS market orders, Toss conditional orders (SINGLE/OCO/OTO — Korean 예약 매수), and Toss USD amount-based US stock orders from a rebalance plan or deposit-allocation list. Use when the user asks to place trades — "리밸런싱 지시서대로 주문 넣어줘", "TOSS 계좌 매매 실행해줘", "ISA 매매 해줘", "예치금 매매해줘", "예약 매수", "조건부 주문", "달러로 미국주식 사줘", "금액 주문", "장이 닫혔는데 주문 걸어줘". Sells finish before buys; every live action needs a dry-run preview and explicit confirmation. Programmatic support: ISA + TOSS only; conditional and USD-amount orders are Toss-only.
---

# Execute Rebalance Plan

Places real market orders for the TOSS and ISA legs of an already-written rebalance-plan
document, via `go run ./cmd/rebalance-order`. This reopens a narrow slice of the automated
execution that ADR-0001 (2026-07-03) deliberately removed in favor of manual execution — do
not extend it to other accounts or to limit/price orders without the user explicitly asking
for that scope change, since ADR-0001's "manual execution" is still the default everywhere
this skill doesn't cover.

## Defaults — do not ask unless overridden

- Ordinary TOSS/ISA execution uses market orders. Do not ask market vs limit. The one
  exception is a **KIS buy sized to fit available cash**: a KIS market buy reserves buying power
  at the daily upper-limit price, so it can be rejected with "주문가능금액을 초과 했습니다" even
  when cash covers the actual fill. In that case place a **limit order** via `-price` (see the
  per-order loop) — this reserves exactly price×qty. Don't ask; switch to a marketable limit
  (current price, or +1 tick to guarantee the fill) automatically and say so.
- Toss USD amount orders use `MARKET`. Do not ask order type; no expiry applies because this is
  an immediate US regular-session order, not a conditional order.
- Toss conditional-order creation defaults order type to `MARKET` for `SINGLE`, `LIMIT` for
  `OCO`/`OTO` (the API requires LIMIT for those), and expiry to KST tomorrow's calendar date. Do
  not ask either value; state both in the dry-run preview. A user-provided order type or expiry
  overrides the defaults.
- Ask only for genuinely missing trade intent: symbol, quantity/amount, and trigger price.
  Reuse values already resolved in the conversation.
- Never default away the final live-action confirmation. A Toss conditional order can fire
  later without a human present, so preview the complete request and require one explicit
  confirmation before `-yes`.

## Toss USD amount-order mode

This mode does not require a rebalance-plan document. Use it to buy or sell a US ticker by an
exact dollar amount; Toss determines fractional share quantity at execution time.

1. Resolve the US ticker, side, and USD amount. If the user says to use available USD, read live
   buying power with `go run ./cmd/pm toss buying-power -account TOSS -currency USD`.
2. State the environment before touching anything: `toss-order-manage` prints the resolved Toss
   base URL to stderr on every invocation as a live-money warning. Unlike KIS, Toss has no
   demo/live split to route between — there's one production URL, only overridable by
   `TOSS_BASE_URL` in `.env` — so this banner is confirming "nothing overrode the default,"
   not choosing between environments. Show it to the user once before any dry-run or live call;
   if `TOSS_BASE_URL` is set to something other than the default, stop and confirm that's
   intentional before proceeding (see kis-debug §13 if the value looks wrong).
3. Confirm the US regular session is open with `pm toss market-calendar-us`; amount orders are
   rejected outside regular hours. Do not represent this as a queued or conditional order.
4. Preview without `-yes`:
   ```bash
   go run ./cmd/toss-order-manage -account TOSS -action create-amount \
     -symbol <ticker> -side BUY -order-amount <usd>
   ```
   Use one stable `-client-order-id` in both preview and live calls when retry safety matters.
5. Show ticker, side, exact USD amount, `MARKET`, and the regular-hours-only constraint (the
   base URL was already shown in step 2). Require explicit confirmation for this exact order.
6. Rerun the identical command with `-yes`. Read `orderId`, then verify it with:
   ```bash
   go run ./cmd/pm toss order -account TOSS -order-id <id>
   ```
   Report the broker's returned status; do not equate order acceptance with full execution.

## Toss conditional order mode (SINGLE / OCO / OTO)

This mode does not require a rebalance-plan document.

1. Resolve symbol, quantity, and trigger price(s). Apply the defaults above without questions —
   for `OCO`/`OTO`, that also means a second leg (`-second-side`, `-second-trigger-price`, and,
   because the default order type is `LIMIT` for these two types, `-first-order-price` /
   `-second-order-price` for each leg's limit price).
2. State the environment before touching anything — same as USD amount-order mode step 2:
   show the Toss base-URL banner once before any dry-run or live call, and stop if
   `TOSS_BASE_URL` looks unexpectedly overridden.
3. Preview without `-yes`; omit the defaulted flags so the CLI itself supplies them.

   Single price-triggered buy:
   ```bash
   go run ./cmd/toss-order-manage -account TOSS -action create-conditional \
     -symbol <ticker> -type SINGLE -quantity <n> \
     -first-side BUY -first-trigger-price <price>
   ```

   OCO (two exit legs on an existing position, e.g. take-profit + stop-loss — both require an
   explicit limit price since the type defaults to `LIMIT`):
   ```bash
   go run ./cmd/toss-order-manage -account TOSS -action create-conditional \
     -symbol <ticker> -type OCO -quantity <n> \
     -first-side SELL -first-trigger-price <take_profit_trigger> -first-order-price <take_profit_limit> \
     -second-side SELL -second-trigger-price <stop_loss_trigger> -second-order-price <stop_loss_limit>
   ```
4. Show symbol, quantity, trigger price(s) (both legs for OCO/OTO), `LIMIT`/`MARKET` per the
   defaults, computed expiry — the base URL was already shown in step 2. Require explicit
   confirmation for this exact order.
5. After explicit confirmation, rerun the identical command with `-yes`.
6. Read `conditionalOrderId` from the response, then verify it with:
   ```bash
   go run ./cmd/pm toss conditional-order -account TOSS \
     -conditional-order-id <id>
   ```
   Report success only when the returned status is `WATCHING`.

**Scope, hard limit:** only accounts whose plan section header names TOSS or ISA get executed.
연금저축, 여유금, or any other account section is always reported as "수동 실행 필요" — never run
`rebalance-order` against them, even if the user asks in the same breath, without a separate
explicit confirmation that the scope should widen.

## Prerequisites (check before anything else)

- For rebalance execution, the plan document must already exist and have passed
  `rebalance-plan`'s step-4 verify script.
  Never re-derive or eyeball trades from memory — read them from the written, verified document.
  If no `.data/rebalance-plan-YYYY-MM.md` exists for the run the user means, tell them to run
  the `rebalance-plan` skill first.
- ISA needs `KisAccountNo` + (optionally) `KisAPIKeyID` set on that account in the app (계좌 관리
  화면); TOSS needs `TossAccountSeq` set, plus `TOSS_CLIENT_ID`/`TOSS_CLIENT_SECRET` in `.env`.
  If a later step errors "account not linked to KIS or Toss" or "KIS is not configured", that's
  what's missing — tell the user to configure it in the app, don't try to work around it.

## Workflow

### 1. Read the plan document

Read `.data/rebalance-plan-YYYY-MM.md` (the month the user means; ask if ambiguous). From §2
매매 지시, pull only the TOSS and ISA account sub-sections. For each order line, extract:

- ticker code (the `(...)` code in "종목명 (코드)"; for a plain overseas ticker like `AAPL`,
  that's the ticker as-is)
- side (매도/매수 sub-table)
- quantity (share count)
- account (ISA or TOSS)

A line marked "주문 시점 재계산" (new position, no share count — see rebalance-plan §3) has no
executable quantity. Skip it and tell the user it needs a manual share-count decision at order
time; never guess a quantity for it.

### 2. State the environment before touching anything

`go run ./cmd/rebalance-order` prints the resolved `KIS_ENV` to stderr on every invocation,
dry-run or not — run one call first (any TOSS/ISA order, without `-yes`) and show the user that
banner verbatim before proceeding. Three cases:

- `real` — live trading, real money. This is the expected case for an actual execution run.
- `demo`/`vps`/`paper` — no live order will be placed. Fine for a rehearsal, but if the user
  meant to actually trade, stop and flag the mismatch before running anything with `-yes`.
- anything else (typo, unset in an unexpected way) — KIS silently routes this to the **real**
  production API (documented footgun, see AGENTS.md). Treat this exactly like the `real` case
  for risk purposes, but tell the user their `KIS_ENV` value looks wrong.

### 3. Sells first, fully — then buys

Rebalance-plan's cash rule is per account: `buys ≤ sells + existing cash`. A buy placed before
its account's sells have actually gone through is placed against cash that may not be there yet.
So split into two hard phases, never interleaved:

**Phase A — all sell orders**, across both accounts, each with its own preview/confirm/execute
(see the per-order loop below). Finish every sell line — success, failure, or user-directed
skip — before starting Phase B.

**Phase B — all buy orders.** Before Phase B starts, re-check each account against what Phase A
actually delivered, not what the plan assumed. `OrderExecutionRecord.Status == "success"` proves
the broker *accepted* the order, not that proceeds settled at a known price — the record carries
no fill price or amount, so don't derive `sell_proceeds_krw` by multiplying the plan's estimated
price by quantity for successful lines. Instead, for each account that had a sell in Phase A, run
`go run ./cmd/pm sync -account "<name>"` (portfolio-sync skill) to pull the broker's actual
post-sell cash balance, and use that synced value as the account's `existing_cash_krw` input
below with `sell_proceeds_krw: 0` (the sync already folds proceeds into cash — don't add them a
second time). An account with no sells this run doesn't need a re-sync; use its existing cash
as-is.
Sum the buy lines still to execute per account, then run:

```bash
python3 .claude/skills/execute-rebalance-plan/scripts/check_cash_availability.py --file <accounts.json>
```

`PASS` — every account's buys proceed as written. `FAIL` names the account and the shortfall
amount; tell the user which buys in that account are at risk of exceeding available cash, and ask
whether to reduce/drop specific buy lines or proceed anyway (brokerage buying power may already
cover it) — never silently shrink or silently execute an oversized buy based on the script's
verdict alone. An account with no sells this run isn't affected by another account's outcome —
cash doesn't move between accounts, so it simply reports `PASS` off its existing cash alone; don't
hold its buys up unnecessarily.

**Synced cash ≠ orderable cash (KIS).** The synced `CashBalance` can include unsettled sell
proceeds and other amounts KIS won't let you spend yet, so `check_cash_availability.py` passing
off the synced number does not guarantee the broker will accept the buy. For a KIS account, read
the real figure before a large buy with:

```bash
go run ./cmd/pm kis order-cash -account "<ISA|여유금>" -ticker <code> -price <limit>
```

`orderable_cash` is the spendable cash; with `-ticker`/`-price` it also returns `max_buy_qty`
(passing `-price` sizes it for a **limit** order at that price, which is what you'll actually
place). Size the buy to `max_buy_qty` rather than to the synced balance. This is KIS-only —
Toss buys wait on their own settlement/FX and `kis order-cash` doesn't apply.

**Per-order loop** (used identically in both phases):

1. Run the dry-run (default, no `-yes`):
   ```bash
   go run ./cmd/rebalance-order -account "<ISA|TOSS>" -ticker <code> -side <buy|sell> -qty <n> -currency <KRW|USD> [-price <limit>]
   ```
   (Add `-exchange NASD|NYSE|AMEX` only if the plan names an actual overseas exchange for a
   KIS-routed ticker — most ISA holdings are KR-listed ETFs and TOSS orders don't need it, Toss
   doesn't take an exchange parameter at all.) Omit `-price` for a market order; add it to place
   a KIS limit order (see the Defaults note — used when a market buy won't fit orderable cash).
   `-price` is KIS-only; it is rejected for Toss accounts. The dry-run preview prints
   `type=market` or `type=limit @ <price>` so you can confirm which was resolved.
2. Show the user exactly what will be sent: account, ticker, side, quantity, order type
   (market, or limit @ price), and the KIS_ENV banner. For a market order remind them there's no
   limit price or partial-fill control; for a limit order state the price and that only the
   marketable portion fills.
3. Wait for an explicit yes for *this specific order* before doing anything else. A blanket
   "다 실행해" up front does not count as per-order confirmation — still confirm each one,
   since a single wrong line (wrong side, wrong quantity) is real money and each is independent.
4. Only on explicit yes, rerun the identical command with `-yes` added. Parse the JSON on
   stdout (an `OrderExecutionRecord`: `Status` is `"success"` or `"failed"`, `Message` carries
   the KIS/Toss error text on failure — it's already persisted to `order_executions` either way,
   so nothing here needs re-logging by hand).
5. Report the result immediately (don't batch reporting until the end — if order 3 of 7 fails,
   the user needs to know before deciding whether to continue). On `"failed"`, stop and ask
   whether to retry, skip this line, or abort the remaining orders — never auto-retry or
   auto-skip silently.

### 4. Close out

After the TOSS/ISA orders are done (fully or partially), update the plan document:

- Change its 상태 field to reflect what happened (e.g. "TOSS/ISA 실행 완료, 연금저축 수동 실행 필요"
  — never claim full execution if any account's orders are still manual-only by scope, or if
  any order failed/was skipped).
- Leave 연금저축 (and any other out-of-scope account) exactly as the plan wrote it — this skill
  never touches that section, so don't imply it did.
- Summarize for the user: how many orders placed successfully, how many failed (with the
  message), how many skipped (new positions with no quantity), and which accounts still need
  manual execution.

## Deposit-only execution mode (buys, no sell phase)

When the input is a deposit allocation from `rebalance-plan`'s deposit mode (new money, buys
only — see that skill's §Deposit mode) rather than a full quarterly plan, there's nothing to
sequence: there are no sells, so Phase A doesn't exist and the cash-shortfall re-check in Phase
B is moot (the deposit itself is the cash). Skip straight to the per-order loop in step 3, run
against TOSS/ISA lines only, exactly as above.

The deposit list may not be a saved file — `rebalance-plan`'s deposit mode usually outputs an
inline table in conversation rather than writing `.data/rebalance-plan-YYYY-MM.md` (a file per
deposit is clutter, per that skill's own guidance). So:

- If the buy list was just produced earlier in this conversation, read it from there — don't
  ask the user to restate it, but do restate it back to them before executing so they can catch
  a transcription slip.
- If the user points at a saved document instead, read it the normal way (step 1).
- Same scope limit applies: only the TOSS/ISA lines are executable here; if the deposit landed
  in 연금저축 or another out-of-scope account, say so and stop — that one's manual regardless of
  how simple a buys-only order is.
- No plan-document status update is needed unless a file exists to update (see step 4) — a
  one-off deposit run just needs the per-order success/failure summary.

## Failure modes to avoid

- Executing an order the user hasn't individually confirmed, even if a batch of similar orders
  was already approved — each order is independent money movement.
- Interleaving a buy with sells still in flight, or starting Phase B without checking whether
  Phase A's sells actually delivered the cash the plan assumed.
- Computing the Phase B cash re-check by hand instead of running `check_cash_availability.py`.
- Re-submitting a KIS market buy at the same quantity after a "주문가능금액 초과" rejection
  instead of switching to a `-price` limit order — the market order reserves at the upper limit,
  so retrying it unchanged just fails again.
- Sizing a large KIS buy off the synced `CashBalance` instead of `pm kis order-cash`'s
  `orderable_cash` / `max_buy_qty`, when the synced number may include unsettled proceeds.
- Running against 연금저축 or any account outside TOSS/ISA scope.
- Guessing a quantity for a "주문 시점 재계산" new-position line instead of skipping it.
- Proceeding past a `"failed"` result without telling the user and asking how to continue.
- Treating an unrecognized `KIS_ENV` value as harmless — it silently means real trading.
- Re-deriving trades from the user's description instead of reading the verified plan document.
