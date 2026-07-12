---
name: portfolio-data
description: >-
  Create, read, update, and delete accounts, groups, stocks, deposits, and
  holdings in the portfolio-manager DB, and show the portfolio dashboard —
  via `go run ./cmd/pm`, the CLI that replaced the app's web UI. Use whenever
  the user wants to manage portfolio data directly: "계좌 추가해줘", "계좌 목록
  보여줘", "예수금 얼마로 바꿔줘", "입금 기록해줘", "이번 달 적립금 입력", "종목 추가해줘",
  "종목을 다른 그룹으로 옮겨줘", "보유수량 수정해줘", "이 종목 삭제해줘", "그룹 목표비중 바꿔줘",
  "지금 포트폴리오 어때?", "대시보드 보여줘", "총 자산 얼마야", or any request to view or edit
  accounts/groups/stocks/deposits/holdings that isn't about rebalancing trades
  (use rebalance-plan/execute-rebalance-plan for those) or broker sync/price
  refresh (use portfolio-sync for those).
---

# Portfolio Data

Thin conversational wrapper around `cmd/pm`'s CRUD and dashboard subcommands. Every call prints
indented JSON to stdout and exits non-zero with a message on stderr on failure — parse that,
don't invent fields. Run all commands from the repo root: `go run ./cmd/pm <resource> <verb> [flags]`.

## Resources and verbs

```
account   list | get -id | add -name -cash | update -id [-name -cash -kis-account-no -kis-api-key-id -account-type -toss-account-seq] | delete -id | set-cash -id -cash
group     list | get -id | add -name -target | update -id [-name -target] | delete -id
stock     list [-group] | get -id | add -group -ticker | update -id [-ticker -exchange -name -asset-class -security-group] | move -id -group | delete -id
deposit   list | get -id | add -amount -date [-note] | update -id [-amount -date -note] | delete -id
holding   list [-account] | get -id | add -account -stock -qty | add-by-ticker -account -ticker -qty | bulk -account -updates | update -id -qty | delete -id
price     list -ticker T [-from DATE -to DATE -limit N] | set -ticker T -date DATE -price P [-currency C -name N -exchange E] | delete -ticker T -date DATE
order-execution list [-limit N] [-ticker T] [-status STATUS]
dashboard [-no-change-rates]
```

`go run ./cmd/pm help` prints this same table if you need to double check a flag name.

## Before you can update/delete/move anything, you need its `-id`

Every `get`/`update`/`delete`/`move` verb takes a UUID `-id`, not a name. Run the matching
`list` first, find the row, and take its `"ID"` field. Use `get -id` to re-check one target
without re-reading the full collection. Never guess or fabricate a UUID.

## Reference resolution — name vs UUID

- `-account` (on `holding` and, indirectly, via account matching elsewhere) accepts an **exact
  account name or a unique case-insensitive substring** — "ISA", "isa", "토스" all work as long
  as they match exactly one account. If the CLI errors "account name %q is ambiguous: matches
  ...", read the list back to the user and ask which one they meant — never guess.
- `-group` (on `stock add`/`stock list`/`stock move`) accepts **either a group UUID or an exact
  group name** — "국내성장" works directly, no need to look up its ID first.

## Field conventions

- Money/quantity fields (`-cash`, `-amount`, `-qty`) are decimal strings — pass `"1234.56"`,
  not a float, and never pre-round what the user gave you.
- Dates (`-date` on `deposit`) are ISO `YYYY-MM-DD`.
- Tickers are case-insensitive on input — `cmd/pm` upper-cases them itself.
- `account-type` is one of `brokerage|irp|pension|isa`; anything else is rejected.
- On `account update`, `-kis-account-no /clear`, `-kis-api-key-id /clear`,
  `-account-type /clear`, and `-toss-account-seq /clear` reset the corresponding nullable
  broker metadata. Clear each field explicitly; clearing one does not silently clear another.
- Account JSON exposes `KisAPIKeyConfigured` and a bounded label such as `KisAPIKeySlot:
  "slot-2"`. It never emits the source key-slot integer or credentials; unknown values become
  `"unmapped"`.
- `pm deposit update -id X -note "/clear"` nulls an existing note. An empty/omitted `-note` on
  `update` leaves the existing note untouched — it is *not* the same as clearing it.
- `update`/`account`/`group`/`stock` verbs only touch the flags you actually pass — every other
  field on the row is preserved as-is. Don't re-pass unrelated fields "just in case".
- `holding bulk -updates "id1:qty1,id2:qty2"` takes a comma-separated list of `holdingID:qty`
  pairs, all against the same `-account`.
- `holding add-by-ticker` looks the ticker up via `stock list`/`GetByTicker` under the hood; if
  the stock doesn't exist yet, it errors rather than guessing a group to create it in — run
  `stock add -group <ref> -ticker <t>` first, then retry.
- `holding list` without `-account` returns all accounts. Holding list/get output includes the
  account name, ticker, stock name, group ID, and group name alongside the stored IDs.
- `price set` corrects one cached daily close. For a new `(ticker,date)` row, `-currency` and
  `-name` are required; when updating an existing row, omitted metadata is preserved.
- `order-execution list` is read-only and intentionally omits the broker `RawResponse` payload.

## Presenting results

Don't dump raw JSON at the user. Summarize in Korean: for a list, a short table or bullet list
with the fields that matter (name, cash, ticker, group, quantity...); for a single
create/update/delete, confirm what changed in one line. Keep the underlying JSON around in case
the user needs to reference an ID next.

## Confirm before destructive actions

`delete` calls (especially `account delete`, which cascades and removes that account's
holdings, and `price delete`, which removes one cached close) are not reversible from the CLI. Confirm with the user before running any
`delete` — state what will be removed — the same way you would before any other irreversible
action. `update`/`move`/`add` are low-risk (still editable afterward) and don't need the same
level of ceremony, but do restate what you're about to run if the request was ambiguous.

## Failure modes to avoid

- Don't fabricate a `-id` because you "remember" it from earlier in the conversation — DB state
  may have changed since; re-list if there's any doubt.
- Don't retry a failed command with different flags hoping something sticks — read the error
  message (it's usually exactly what's wrong: invalid UUID, missing required flag, ambiguous
  account name, invalid account-type) and fix that specific thing.
- A delete of a missing ID/date returns an error. Never report deletion unless the command
  itself succeeds.
- `dashboard` requires KIS-backed price data to compute the full summary; if it falls back to a
  per-group holdings breakdown (no `TotalValue`/`ReturnRate` fields), prices may be missing or
  stale — point the user at the portfolio-sync skill's `price-sync` rather than trying to
  compute totals yourself from the holdings list.
