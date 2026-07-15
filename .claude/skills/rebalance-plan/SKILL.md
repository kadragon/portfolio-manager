---
name: rebalance-plan
description: Monitor the portfolio monthly and produce threshold-triggered, per-account trade instructions from the portfolio-manager DB using .data/rebalance-policy.md. Use for rebalancing plans, trade instructions, drift checks, monthly checks, or "리밸런싱", "지시서", "포트폴리오 점검", "월간 점검", "지금 비중 얼마나 틀어졌어?". Also for buy-only deposit allocation ("예치금 들어왔어", "이번 달 적립금 배분"), policy revisions (target/band/group changes), gradual target transitions, and the late-January annual review of targets, risk capacity, tax rules, and account placement. Monthly monitoring never forces trades; glide paths advance only in Feb/May/Aug/Nov.
---

# Rebalance Plan

Generate a per-account trade instruction document. The policy (targets, tax rules, bands) lives in
`.data/rebalance-policy.md` — read it first, every run. This skill is the engine; the policy file is the data.

## Workflow

### 1. Gather inputs

- Read `.data/rebalance-policy.md`. If missing, stop and tell the user — never invent targets.
- Check policy §Transition Schedule. Any group listed there is mid phased-change (see
  `references/policy-revision.md` §Gradual transition mode) — its **effective target this run is the interim value**,
  not the §Target Allocation value. State which groups are mid-transition and the interim
  target in effect before computing anything.
- Get current USD/KRW (WebSearch or ask). State the rate used; never silently reuse an old one.
- Run the snapshot script from the repo root:

```bash
python3 .claude/skills/rebalance-plan/scripts/snapshot.py --fx <rate>
```

- Surface every `warnings` entry (stale prices, missing prices, non-positive prices) to the
  user before planning. A stale price on a ticker the plan will trade → stop, run
  `go run ./cmd/pm price-sync` (or invoke the portfolio-sync skill) yourself, then re-run the
  snapshot. Stale price on an untraded ticker → proceed and note it. A **missing price**
  (`no price for <ticker>`, snapshot has no row at all) is treated exactly like a non-positive
  price, not like a stale one — it always stops regardless of whether that ticker trades this
  run, same reason: the snapshot zeroes the holding's value, which understates the whole group
  and can manufacture a phantom underweight elsewhere in the plan. A **non-positive price**
  warning always stops regardless of whether that ticker trades this run — the snapshot
  silently zeroes the holding's value, which understates the whole group and can manufacture a
  phantom underweight elsewhere in the plan.
- If snapshot `db_target_pct` disagrees with policy targets, policy wins — but report the
  discrepancy so the user updates the app's group settings.

### 2. Compute deviations

For each group, gather its band from policy §Trading Rules (per-group; never assume a default)
and its effective target — the §Target Allocation value, or the interim value from an active
§Transition Schedule entry (see step 1) — same basis for both, so a group already at its interim
value reads as in-band, not as a stale-target miss. `weight_pct` (from the snapshot) is already
computed against total portfolio value including cash — use it as-is.

Glide-path advancement runs unconditionally as part of this step, not only when the outcome
turns out to be no-trade: if this month is a scheduled advance month (February, May, August, or
November), perform any due §Transition Schedule advancement under `references/policy-revision.md`
§Gradual transition mode step 4 **before** computing deviations, so every deviation, band check,
and downstream trade decision this run — breach or no breach — is against the advanced interim
targets, never the stale pre-advancement ones.

Build one JSON object per group (`name`, `weight_pct`, `effective_target_pct`, `band_pct`, and
`hard_ceiling_pct` only for a group carrying a mechanical override — currently just the KOSPI
exit rule, 국내성장 ≥ 30%, which also overrides the "prefer new money over selling" preference)
and run:

```bash
python3 .claude/skills/rebalance-plan/scripts/check_deviations.py --file <groups.json>
```

Trust its `NO-TRADE`/`TRADE-NEEDED` call over hand arithmetic — a KRW-scale rounding slip is
exactly the kind of error that turns a true in-band group into a false breach. Separately,
identify placement violations yourself (an asset sitting in an account the policy forbids, e.g.
KR-listed overseas ETF in TOSS — this needs policy semantics, not group-level numbers) and pass
each one via `--placement-violation "<description>"` (repeatable) so it lands in the same report
as a forced trade regardless of group deviation.

If the report says no groups need trades and no placement violation was passed, write no
document — report "이번 점검 매매 불필요" with the group-deviation table, then stop. A breach
starts a trade decision; it does not override the policy's tax-aware deferral rules. If the only
breach is an underweight that a **confirmed** contribution before the next monthly monitoring run
can fully bring inside band, write no trade document: report the amount and date required, and
re-check after the deposit. Never use an assumed or unscheduled contribution to defer a breach.
KOSPI exit and placement violations cannot be deferred.

### 3. Design trades

Work account by account. Hard constraints from policy §Tax Placement Rules — the important ones:

- Cash cannot move between accounts: each account's buys ≤ its sells + existing cash.
- 연금저축 holds defensive only; ISA prefers KR-listed overseas (domestic-equity also allowed);
  TOSS holds US-listed + domestic-equity. "Prefers" is not a violation — only policy's banned
  combos count as placement violations.
- Never plan US-listed sells without flagging the estimated gain (양도세 22%) and asking.
- KR-listed overseas ETF sells in the taxable account are taxable — flag the amount; prefer ISA/연금 sells.
- 여유금: no trades. Prefer covering underweights with scheduled new contributions over selling.
- Apply the policy's trade waterfall: new cash and distributions first, then tax-sheltered
  account trades, then tax-free domestic-equity adjustments, and taxable sales last.
- Apply the policy's destination rule. Overweight taxable positions stop at target + half-band;
  deposit buys fill toward target; tax-sheltered positions may reach target. The KOSPI exit
  still trims directly to 20%.

Round to whole shares for KRW-listed ETFs; USD ETFs may be stated as amounts plus an FX-conversion
step. Skipping a sub-1-share buy is fine — say so.

**New positions** (ticker absent from DB): the snapshot has no price for them. State those trades
as KRW amounts, not share counts — a web quote is provisional at best, so mark quantities
"주문 시점 재계산" and tell the user to register the ticker (`pm stock add`, via the portfolio-data
skill) and run `pm price-sync` so the next run has real prices. Never present a share count
derived from a stale web quote as executable.

### 4. Verify before writing (required)

Build the trade plan's numbers into the JSON shape documented in
`scripts/verify_plan.py`'s docstring (accounts: pre value/cash/buys/sells; groups: pre
value/net trade/effective target %/band %p — effective = interim target if the group is
transitioning) and run:

```bash
python3 .claude/skills/rebalance-plan/scripts/verify_plan.py --file <plan.json>
```

It checks, in `Decimal` (float drift on won-scale sums is exactly the kind of error this
catches and an inline ad-hoc check might not):

1. Per account: `sum(buys) ≤ sum(sells) + cash`, remainder < 1% of account value.
2. Post-trade group weights within band of the effective target from step 2.
3. Account-level net trade total equals group-level net trade total (no money invented or
   double-counted).

A `FAIL` line names the specific violation — fix and re-run, or if it's an accepted exception
(e.g. remainder parked for a planned US-listed buy), state the exception in the document
instead of suppressing the check. A plan that still fails never reaches the document.

### 5. Write the document

Path: `.data/rebalance-plan-YYYY-MM.md` (run month). Korean, user-facing. Structure:

```markdown
# 포트폴리오 리밸런싱 지시서 YYYY-MM
- 기준일 / 환율 / 총평가액 / 상태: 미실행
## 1. 현재 vs 목표 (그룹 편차 표, 밴드 내 그룹은 "무거래" 표기)
## 2. 매매 지시 (계좌별 표: 구분/종목/수량/금액/세금 메모)
## 3. 무거래 항목과 이유
## 4. 세금 이벤트 요약 (과세 매도 차익 추정, 확인 필요 항목)
## 5. 앱 반영 순서
```

End by telling the user: document path, order count, total taxable events, and the single largest trade.

## Deposit mode (new-money allocation)

When the user reports a deposit (amount + account) instead of asking for a full rebalance,
plan **buys only — never sells**. New money is the cheapest rebalancing there is: it fixes
underweights without tax events, which is why the policy prefers it over selling.

1. Same inputs as step 1 (policy, FX, snapshot). Ask for the account if not given —
   placement rules differ per account (e.g. cash landing in 연금저축 may only buy 금·채권).
   If this month is a scheduled advance month (February, May, August, or November) and no
   full-plan run has already advanced the schedule this month, perform any due §Transition
   Schedule advancement under `references/policy-revision.md` §Gradual transition mode step 4
   first — same as full-plan mode step 2 — so a deposit-only run doesn't leave the schedule
   stale for the cycle.
2. Recompute group weights against `total + deposit`. Allocate the deposit to below-target
   groups in proportion to their shortfall (%p × total value), filling toward the effective
   target — the §Target Allocation value, or the interim value if the group has an active
   §Transition Schedule entry (new money steers toward this quarter's interim target, never
   ahead of the glide path). The trading band does NOT gate deposit allocation — new money
   always deploys, even at 0.3%p drift; the band only exists to stop tax-costly sells.
3. Only groups purchasable in the deposited account (policy placement rules) are eligible.
   If the account's eligible groups are all at/above target, say so and suggest which
   account the money should have gone to.
4. Round to whole shares; leftover stays as cash and is carried in the report.
5. Verify: `sum(buys) ≤ deposit + account's existing cash`. Output can be an inline table
   (no document file) unless the user asks for one — deposits recur monthly and a file per
   deposit is clutter.

## Policy changes, annual review, and phased transitions

These are not part of routine monthly monitoring or deposit allocation — read
`references/policy-revision.md` in full when any of the following applies:

- The user proposes changing targets, bands, or placement rules ("목표 비중 바꾸자", "리밸런싱
  기준 수정", "금 비중 줄일까?") → **Policy revision mode**.
- It's the late-January annual review, or an earlier material change in horizon/income/tax/risk
  capacity warrants one → **Annual policy review mode**.
- A target change exceeds 5%p and should be phased in over quarterly cycles instead of applied
  at once, or the user asks directly ("점진적으로 바꾸자") → **Gradual transition mode**.

## Failure modes to avoid

- Planning from memory of a previous quarter instead of a fresh snapshot.
- Computing band/deviation by hand instead of running `check_deviations.py`.
- Using the app DB's group targets when the policy file says otherwise.
- Letting a buy exceed an account's sell proceeds (cash is account-local).
- Hiding a taxable event inside a routine trade list — tax notes are per-line, not a footnote.
- Applying a large target change (>5%p) in one shot without offering `references/policy-revision.md`
  §Gradual transition mode.
- Forgetting to advance §Transition Schedule in February, May, August, or November even when
  no trade document is written — leaves the next cycle using a stale interim target.
- Advancing §Transition Schedule during an ordinary monthly check outside February, May,
  August, or November.
- Treating the late-January policy review as a mandatory rebalance or target change.
