---
name: rebalance-plan
description: Produce a quarterly (Feb/May/Aug/Nov) portfolio rebalancing trade-instruction document from the portfolio-manager DB, applying the user's allocation targets and Korean tax placement rules from .data/rebalance-policy.md. Use whenever the user asks for a rebalancing plan, trade instructions, portfolio drift check, or says "리밸런싱", "리밸런스 계획", "지시서", "포트폴리오 점검", "분기 리밸런싱", or mentions rebalancing in February, May, August, or November — even without naming the skill. Also use for off-cycle checks like "지금 비중 얼마나 틀어졌어?".
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

Per group: `deviation = weight_pct − policy target`. Apply the policy band (default ±1.5%p):
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
- 연금저축 holds defensive only; ISA holds KR-listed overseas; TOSS holds US-listed + domestic-equity.
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

1. Per account: `sum(buys) ≤ sum(sells) + cash` and unused remainder < 1% of account value.
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

## Failure modes to avoid

- Planning from memory of a previous quarter instead of a fresh snapshot.
- Using the app DB's group targets when the policy file says otherwise.
- Letting a buy exceed an account's sell proceeds (cash is account-local).
- Hiding a taxable event inside a routine trade list — tax notes are per-line, not a footnote.
