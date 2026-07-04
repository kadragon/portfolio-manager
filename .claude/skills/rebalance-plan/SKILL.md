---
name: rebalance-plan
description: Produce a quarterly (Feb/May/Aug/Nov) portfolio rebalancing trade-instruction document from the portfolio-manager DB, applying the user's allocation targets and Korean tax placement rules from .data/rebalance-policy.md. Use whenever the user asks for a rebalancing plan, trade instructions, portfolio drift check, or says "리밸런싱", "리밸런스 계획", "지시서", "포트폴리오 점검", "분기 리밸런싱", or mentions rebalancing in February, May, August, or November — even without naming the skill. Also use for off-cycle checks like "지금 비중 얼마나 틀어졌어?" and for deposit allocation — when the user says new money arrived or asks what to buy with a deposit ("예치금 들어왔어", "500만원 입금했는데 뭐 살까", "이번 달 적립금 배분"). Also use when the user wants to change the allocation policy itself — new targets, band changes, adding/removing groups ("목표 비중 바꾸자", "리밸런싱 기준 수정", "금 비중 줄일까?") — revision mode checks proposals against the recorded design rationale before anything changes.
---

# Rebalance Plan

Generate a per-account trade instruction document. The policy (targets, tax rules, bands) lives in
`.data/rebalance-policy.md` — read it first, every run. This skill is the engine; the policy file is the data.

## Workflow

### 1. Gather inputs

- Read `.data/rebalance-policy.md`. If missing, stop and tell the user — never invent targets.
- Get current USD/KRW (WebSearch or ask). State the rate used; never silently reuse an old one.
- Run the snapshot script from the repo root:

```bash
python3 .claude/skills/rebalance-plan/scripts/snapshot.py --fx <rate>
```

- Surface every `warnings` entry (stale prices, missing prices) to the user before planning.
  A stale price on a ticker the plan will trade → stop, ask the user to run the app's KIS sync,
  then re-run the snapshot. Stale price on an untraded ticker → proceed and note it.
- If snapshot `db_target_pct` disagrees with policy targets, policy wins — but report the
  discrepancy so the user updates the app's group settings.

### 2. Compute deviations

Per group: `deviation = weight_pct − policy target`. `weight_pct` is computed against total
portfolio value including cash — use it as-is, don't recompute against holdings only.
Apply each group's band from policy §Trading Rules (per-group; never assume a default):
groups inside the band get **no trades**. Two triggers override the band:

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

Run a Python check (inline, Bash) proving:

1. Per account: `sum(buys) ≤ sum(sells) + cash` and unused remainder < 1% of account value
   (or explained — e.g. cash parked for a planned US-listed buy).
2. Post-trade group weights within band of targets (or explained, e.g. absorbed remainder).
3. Post-trade totals ≈ pre-trade total (no money invented).

A plan that fails verification never reaches the document. Fix and re-verify.

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
   groups in proportion to their shortfall (%p × total value), filling toward target.
   The trading band does NOT gate deposit allocation — new money always deploys, even at
   0.3%p drift; the band only exists to stop tax-costly sells.
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
3. Sanity-check the resulting policy as a whole: targets sum to 100%, every group has an account
   placement, equity/defensive split recomputed and stated against the cap recorded in
   §Design Rationale (currently 15% defensive — read it from the policy, don't assume).
4. On a confirmed change, update the policy file in both places: the rule itself AND the
   rationale table (new value + new reason), plus a dated entry in §Revision changelog.
   A rule changed without its rationale updated means the next revision inherits a stale "why".
5. The DB is not updated by this skill (snapshot reads groups/targets from the app DB, read-only).
   After a target change — and especially after adding/removing a group, which the step-1
   discrepancy check cannot detect because a markdown-only group has no DB row at all —
   tell the user to mirror the change in the app's group settings, then verify with a fresh
   snapshot that `db_target_pct` agrees.
6. Normal plan runs and deposit runs never write the policy file — only revision mode does.

## Failure modes to avoid

- Planning from memory of a previous quarter instead of a fresh snapshot.
- Using the app DB's group targets when the policy file says otherwise.
- Letting a buy exceed an account's sell proceeds (cash is account-local).
- Hiding a taxable event inside a routine trade list — tax notes are per-line, not a footnote.
