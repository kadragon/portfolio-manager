---
name: rebalance-plan
description: Produce a quarterly (Feb/May/Aug/Nov) portfolio rebalancing trade-instruction document from the portfolio-manager DB, applying the user's allocation targets and Korean tax placement rules from .data/rebalance-policy.md. Use whenever the user asks for a rebalancing plan, trade instructions, portfolio drift check, or says "리밸런싱", "리밸런스 계획", "지시서", "포트폴리오 점검", "분기 리밸런싱", or mentions rebalancing in February, May, August, or November — even without naming the skill. Also use for off-cycle checks like "지금 비중 얼마나 틀어졌어?" and for deposit allocation — when the user says new money arrived or asks what to buy with a deposit ("예치금 들어왔어", "500만원 입금했는데 뭐 살까", "이번 달 적립금 배분"). Also use when the user wants to change the allocation policy itself — new targets, band changes, adding/removing groups ("목표 비중 바꾸자", "리밸런싱 기준 수정", "금 비중 줄일까?") — revision mode checks proposals against the recorded design rationale before anything changes. Also use when a target change is too big to apply in one quarter and the user wants to phase it in gradually ("점진적으로 바꾸자", "한번에 바꾸면 문제되니 나눠서", "금 비중 단계적으로 올려줘") — gradual transition mode spreads the move across several quarterly cycles via an interim glide-path target.
---

# Rebalance Plan

Generate a per-account trade instruction document. The policy (targets, tax rules, bands) lives in
`.data/rebalance-policy.md` — read it first, every run. This skill is the engine; the policy file is the data.

## Workflow

### 1. Gather inputs

- Read `.data/rebalance-policy.md`. If missing, stop and tell the user — never invent targets.
- Check policy §Transition Schedule. Any group listed there is mid phased-change (see
  §Gradual transition mode below) — its **effective target this run is the interim value**,
  not the §Target Allocation value. State which groups are mid-transition and the interim
  target in effect before computing anything.
- Get current USD/KRW (WebSearch or ask). State the rate used; never silently reuse an old one.
- Run the snapshot script from the repo root:

```bash
python3 .claude/skills/rebalance-plan/scripts/snapshot.py --fx <rate>
```

- Surface every `warnings` entry (stale prices, missing prices, non-positive prices) to the
  user before planning. A stale price on a ticker the plan will trade → stop, ask the user to
  run the app's KIS sync, then re-run the snapshot. Stale price on an untraded ticker → proceed
  and note it. A **non-positive price** warning always stops regardless of whether that ticker
  trades this run — the snapshot silently zeroes the holding's value, which understates the
  whole group and can manufacture a phantom underweight elsewhere in the plan.
- If snapshot `db_target_pct` disagrees with policy targets, policy wins — but report the
  discrepancy so the user updates the app's group settings.

### 2. Compute deviations

Per group: `deviation = weight_pct − effective target` (the §Target Allocation value, or the
interim value from an active §Transition Schedule entry — see step 1). `weight_pct` is computed against total
portfolio value including cash — use it as-is, don't recompute against holdings only.
Apply each group's band from policy §Trading Rules (per-group; never assume a default),
computed off the **effective target** (interim, if the group is transitioning — same basis as
the deviation itself, so a group already at its interim value reads as in-band, not as a
5%p miss): groups inside the band get **no trades**. Two triggers override the band:

- KOSPI exit rule (국내성장 ≥ 30%) → mandatory mechanical trim; it also overrides the
  "prefer new money over selling" preference.
- Placement violations (an asset sitting in an account the policy forbids, e.g. KR-listed
  overseas ETF in TOSS) → trade regardless of group deviation.

If nothing breaches the band and no override fires, write no document — report
"이번 분기 매매 불필요" with the deviation table and stop.

### 3. Design trades

Work account by account. Hard constraints from policy §Tax Placement Rules — the important ones:

- Cash cannot move between accounts: each account's buys ≤ its sells + existing cash.
- 연금저축 holds defensive only; ISA prefers KR-listed overseas (domestic-equity also allowed);
  TOSS holds US-listed + domestic-equity. "Prefers" is not a violation — only policy's banned
  combos count as placement violations.
- Never plan US-listed sells without flagging the estimated gain (양도세 22%) and asking.
- KR-listed overseas ETF sells in the taxable account are taxable — flag the amount; prefer ISA/연금 sells.
- 여유금: no trades. Prefer covering underweights with scheduled new contributions over selling.

Round to whole shares for KRW-listed ETFs; USD ETFs may be stated as amounts plus an FX-conversion
step. Skipping a sub-1-share buy is fine — say so.

**New positions** (ticker absent from DB): the snapshot has no price for them. State those trades
as KRW amounts, not share counts — a web quote is provisional at best, so mark quantities
"주문 시점 재계산" and tell the user to register the ticker in the app + run KIS sync so the next
run has real prices. Never present a share count derived from a stale web quote as executable.

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

## Policy revision mode (changing the criteria themselves)

When the user proposes changing targets, bands, placement rules, or drafting new criteria
("목표 비중 바꾸자", "리밸런싱 기준 수정", "금 비중 줄일까?"), the job shifts from executing the
policy to guarding its history:

1. Read policy §Design Rationale first. It records *why* each rule exists (the 2026-07
   five-portfolio review). The policy file is the user's — nothing is forbidden — but no rule
   may be dropped because its reason was forgotten.
2. Diff the proposal against each invariant in that table. For every conflict, present:
   the invariant, its original reason (source portfolio + numbers), and what the change gives up.
   Then ask the user to confirm knowing that. A proposal touching no invariant needs no ceremony.
   For a change that does touch an invariant, add one inversion question before confirming:
   "이 목표가 몇 년 뒤 틀렸다고 판명나는 시나리오는?" — the point isn't to block the change, it's to
   have the failure case on record next to the reason, the same way the KOSPI exit rule exists
   because a tilt without a stated failure condition becomes permanent by default.
3. Sanity-check the resulting policy as a whole: targets sum to 100%, every group has an account
   placement, equity/defensive split recomputed and stated against the cap recorded in
   §Design Rationale (currently 15% defensive — read it from the policy, don't assume).
4. On a confirmed change, update the policy file in both places: the rule itself AND the
   rationale table (new value + new reason, plus the inversion answer from step 2 if one was
   asked), plus a dated entry in §Revision changelog.
   A rule changed without its rationale updated means the next revision inherits a stale "why".
   If a target's delta exceeds 5%p, ask whether to phase it in instead of applying it in one
   shot — see §Gradual transition mode.
5. The DB is not updated by this skill (snapshot reads groups/targets from the app DB, read-only).
   After a target change — and especially after adding/removing a group, which the step-1
   discrepancy check cannot detect because a markdown-only group has no DB row at all —
   tell the user to mirror the change in the app's group settings, then verify with a fresh
   snapshot that `db_target_pct` agrees.
6. Normal plan runs and deposit runs never write the policy file — only revision mode does.

## Gradual transition mode (phased target changes)

A target change of more than 5%p (금 10→20%, e.g.) closes in one shot the moment the band
rule sees it — a lump trade the band can't smooth, and often a lump tax event. Phasing spreads
it over several quarterly cycles as a glide path instead: each run only nudges toward an interim
target, so the trade size and tax hit stay in the same range as an ordinary rebalance.

1. Triggered from Policy revision mode step 4, or any time the user asks directly
   ("점진적으로 바꾸자", "한번에 말고 나눠서"). Ask step size if not given — default **5%p per
   quarter** (matches the band's absolute leg, so each step alone wouldn't have triggered a
   trade under the old target either). A different step is fine; note the tradeoff — bigger
   steps finish sooner but produce bigger per-quarter trades and taxable events.
2. Write the FINAL value to §Target Allocation (that stays the north star). Add a row to
   §Transition Schedule: group, interim target (starts at the old value), final target, step
   %p/quarter, Started (this run's YYYY-MM), Last advanced (leave blank — nothing has advanced
   yet). Log the decision (old → final, step size, reason) in §Revision changelog. If this
   change is one half of a compensating pair (e.g. 금 up / 채권 down to hold 85/15), schedule
   both rows together with the same step cadence — see point 7 below.
3. Every run after that (see step 1/2 of the main workflow), a group in §Transition Schedule
   uses its interim value — not §Target Allocation — for deviation math and deposit
   allocation. Tell the user plainly: "X군 목표 전환 중 — 이번 분기 기준 Y%, 최종 목표 Z%".
4. After the plan is written and verified, advance each active row's interim target one step
   toward final (clamp on the last step — don't overshoot), and update §Transition Schedule in
   the policy file — including its **Last advanced** column (YYYY-MM of this run). When interim
   reaches final, delete the row and log completion in §Revision changelog. This applies
   whether or not the group actually traded this run — the glide path advances on a schedule,
   not on execution (a skipped trade just means next quarter's gap is a bit wider, still
   bounded by the step size). Guard against double-advancing: if **Last advanced** already
   matches this run's YYYY-MM (e.g. an earlier off-cycle KOSPI-exit document already advanced
   it this same month), don't advance again.
5. The KOSPI exit rule and placement violations still override everything, phased or not —
   they're mechanical safety rules, not target-seeking ones.
6. The user can edit an active transition anytime through Policy revision mode: change the
   step, jump straight to final, or cancel back to the pre-transition value — it's a normal
   policy edit, just touching §Transition Schedule instead of §Target Allocation.
7. If two or more groups transition at once (e.g. 금 up, 채권 down, to keep equity/defensive
   at 85/15 throughout), schedule them on the same step cadence so every quarter's full set of
   effective targets still sums to 100% — not just the final ones. If a transition is scheduled
   alone (its interim moving with no offsetting group), the effective-target column in the
   group-deviation table (§5) won't sum to 100% until it completes; note that explicitly in the
   document so it reads as an expected in-progress state, not a data error.

## Failure modes to avoid

- Planning from memory of a previous quarter instead of a fresh snapshot.
- Using the app DB's group targets when the policy file says otherwise.
- Letting a buy exceed an account's sell proceeds (cash is account-local).
- Hiding a taxable event inside a routine trade list — tax notes are per-line, not a footnote.
- Applying a large target change (>5%p) in one shot without offering §Gradual transition mode.
- Forgetting to advance §Transition Schedule after writing the plan — leaves next quarter
  computing deviations against a stale interim target.
