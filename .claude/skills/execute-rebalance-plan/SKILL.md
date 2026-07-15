---
name: execute-rebalance-plan
description: Execute TOSS and ISA orders from a rebalance plan or deposit-allocation list as real market orders, create Toss conditional/reserved orders, and place Toss USD amount-based US stock orders. Use whenever the user asks to place trades — "리밸런싱 지시서대로 주문 넣어줘", "TOSS 계좌 매매 실행해줘", "ISA 매매 해줘", "예치금 매매해줘", "예약 매수", "조건부 주문", "달러로 미국주식 사줘", "금액 주문", or "장이 닫혔는데 주문 걸어줘". Sells finish before buys; every live action needs an exact dry-run preview and explicit final confirmation. Programmatic execution covers ISA and TOSS only; Toss alone supports conditional and USD amount orders.
---

# Execute Rebalance Plan

Places real market orders for the TOSS and ISA legs of an already-written rebalance-plan
document, via `go run ./cmd/rebalance-order`. This reopens a narrow slice of the automated
execution that ADR-0001 (2026-07-03) deliberately removed in favor of manual execution — do
not extend it to other accounts or to limit/price orders without the user explicitly asking
for that scope change, since ADR-0001's "manual execution" is still the default everywhere
this skill doesn't cover.

## Defaults — do not ask unless overridden

- Ordinary TOSS/ISA execution uses market orders. Do not ask market vs limit.
- Toss USD amount orders use `MARKET`. Do not ask order type; no expiry applies because this is
  an immediate US regular-session order, not a conditional order.
- Toss conditional-order creation defaults to `MARKET` and KST tomorrow's calendar date as
  `expireDate`. Do not ask either value; state both in the dry-run preview. A user-provided
  order type or expiry overrides the defaults.
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
2. Confirm the US regular session is open with `pm toss market-calendar-us`; amount orders are
   rejected outside regular hours. Do not represent this as a queued or conditional order.
3. Preview without `-yes`:
   ```bash
   go run ./cmd/toss-order-manage -account TOSS -action create-amount \
     -symbol <ticker> -side BUY -order-amount <usd>
   ```
   Use one stable `-client-order-id` in both preview and live calls when retry safety matters.
4. Show ticker, side, exact USD amount, `MARKET`, the regular-hours-only constraint, and live
   Toss base URL. Require explicit confirmation for this exact order.
5. Rerun the identical command with `-yes`. Read `orderId`, then verify it with:
   ```bash
   go run ./cmd/pm toss order -account TOSS -order-id <id>
   ```
   Report the broker's returned status; do not equate order acceptance with full execution.

## Toss conditional buy mode

This mode does not require a rebalance-plan document. For a single price-triggered buy:

1. Resolve symbol, quantity, and trigger price. Apply the defaults above without questions.
2. Preview without `-yes`; omit the defaulted flags so the CLI itself supplies them:
   ```bash
   go run ./cmd/toss-order-manage -account TOSS -action create-conditional \
     -symbol <ticker> -type SINGLE -quantity <n> \
     -first-side BUY -first-trigger-price <price>
   ```
3. Show symbol, quantity, trigger price, `MARKET`, computed expiry, and live Toss base URL.
4. After explicit confirmation, rerun the identical command with `-yes`.
5. Read `conditionalOrderId` from the response, then verify it with:
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
actually delivered, not what the plan assumed:
- If every sell in an account succeeded as planned, that account's buys proceed as written.
- If a sell in an account failed or was skipped, that account now has less cash than the plan
  assumed. Tell the user which buys in that account are at risk of exceeding available cash,
  and ask whether to reduce/drop specific buy lines or proceed anyway (brokerage buying power
  may already cover it) — never silently shrink or silently execute an oversized buy.
- Buys in an account with no sells this run (or in the other account) aren't affected by another
  account's sell outcome — cash doesn't move between accounts, so don't hold them up unnecessarily.

**Per-order loop** (used identically in both phases):

1. Run the dry-run (default, no `-yes`):
   ```bash
   go run ./cmd/rebalance-order -account "<ISA|TOSS>" -ticker <code> -side <buy|sell> -qty <n> -currency <KRW|USD>
   ```
   (Add `-exchange NASD|NYSE|AMEX` only if the plan names an actual overseas exchange for a
   KIS-routed ticker — most ISA holdings are KR-listed ETFs and TOSS orders don't need it, Toss
   doesn't take an exchange parameter at all.)
2. Show the user exactly what will be sent: account, ticker, side, quantity, and the KIS_ENV
   banner. Remind them it's a **market order** — no limit price, no partial-fill control.
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
- Running against 연금저축 or any account outside TOSS/ISA scope.
- Guessing a quantity for a "주문 시점 재계산" new-position line instead of skipping it.
- Proceeding past a `"failed"` result without telling the user and asking how to continue.
- Treating an unrecognized `KIS_ENV` value as harmless — it silently means real trading.
- Re-deriving trades from the user's description instead of reading the verified plan document.
