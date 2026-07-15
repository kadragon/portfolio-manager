# Policy Revision, Annual Review, and Gradual Transition

Loaded from `SKILL.md` when the user proposes changing rebalancing criteria, at the late-January
annual review, or when phasing in a large target change. Not part of the routine monthly
monitoring or deposit-allocation path.

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
   shot — see §Gradual transition mode below.
5. The DB is not updated by this skill (snapshot reads groups/targets from the app DB, read-only).
   After a target change — and especially after adding/removing a group, which SKILL.md's step-1
   discrepancy check cannot detect because a markdown-only group has no DB row at all —
   tell the user to mirror the change in the app's group settings, then verify with a fresh
   snapshot that `db_target_pct` agrees.
6. Normal plan runs and deposit runs never write the policy file except for scheduled
   §Transition Schedule advancement under §Gradual transition mode step 4; only revision mode
   changes policy criteria.

## Annual policy review mode (late January)

The annual review evaluates the policy; it does not force trades or target changes.

1. Run in late January, or earlier after a material change in investment horizon, income, tax
   treatment, account availability, or risk capacity.
2. Re-read §Profile, §Tax Placement Rules, and every §Design Rationale invariant. Check whether
   the recorded assumptions still match the user's circumstances and current Korean tax/account
   rules; browse authoritative current sources for anything time-sensitive.
3. Report each item as unchanged, needs evidence, or proposed revision. Do not infer a target
   change merely because markets moved.
4. Any proposed revision follows Policy revision mode above, including invariant conflict
   disclosure, rationale update, changelog entry, and transition offer for target changes over 5%p.
5. If no revision is needed, report that result without touching the policy or generating trades.

## Gradual transition mode (phased target changes)

A target change of more than 5%p (금 10→20%, e.g.) closes in one shot the moment the band
rule sees it — a lump trade the band can't smooth, and often a lump tax event. Phasing spreads
it over scheduled quarterly cycles as a glide path instead: each February/May/August/November
cycle only nudges toward an interim target, so the trade size and tax hit stay in the same range
as an ordinary rebalance.

1. Triggered from Policy revision mode step 4 above, or any time the user asks directly
   ("점진적으로 바꾸자", "한번에 말고 나눠서"). Ask step size if not given — default **5%p per
   quarter** (matches the band's absolute leg, so each step alone wouldn't have triggered a
   trade under the old target either). A different step is fine; note the tradeoff — bigger
   steps finish sooner but produce bigger per-quarter trades and taxable events.
2. Write the FINAL value to §Target Allocation (that stays the north star). Add a row to
   §Transition Schedule: group, interim target, final target, step %p/quarter, Started, and
   Last advanced. If the decision occurs in February, May, August, or November, start at the
   **old value advanced one step toward final** (clamped) and set Last advanced to this run's
   YYYY-MM. Otherwise start at the old value and set Last advanced to `—`; the first move occurs
   in the next scheduled advance month. Log the decision (old → final, step size, reason) in
   §Revision changelog. If this change is one half of a compensating pair
   (e.g. 금 up / 채권 down to hold 85/15), schedule both rows together with the same step
   cadence — see point 7 below.
3. Every run after that (see SKILL.md's main workflow steps 1/2), a group in §Transition
   Schedule uses its interim value — not §Target Allocation — for deviation math and deposit
   allocation. Tell the user plainly: "X군 목표 전환 중 — 이번 분기 기준 Y%, 최종 목표 Z%".
4. Advance active rows only during February, May, August, or November. A monthly monitoring run
   in any other month never advances the glide path. During an advance month, move each interim
   target one step toward final (clamp on the last step), even if no trade is needed, and update
   **Last advanced** with that run's YYYY-MM. This schedule action is independent of whether a
   trade document was written. When interim reaches final, delete the row and log completion in
   §Revision changelog. Guard against double-advancing: if **Last advanced** already matches the
   current YYYY-MM, do not advance again.
5. The KOSPI exit rule and placement violations still override everything, phased or not —
   they're mechanical safety rules, not target-seeking ones.
6. The user can edit an active transition anytime through Policy revision mode above: change the
   step, jump straight to final, or cancel back to the pre-transition value — it's a normal
   policy edit, just touching §Transition Schedule instead of §Target Allocation.
7. If two or more groups transition at once (e.g. 금 up, 채권 down, to keep equity/defensive
   at 85/15 throughout), schedule them on the same step cadence so every quarter's full set of
   effective targets still sums to 100% — not just the final ones. If a transition is scheduled
   alone (its interim moving with no offsetting group), the effective-target column in the
   group-deviation table (SKILL.md §5) won't sum to 100% until it completes; note that explicitly
   in the document so it reads as an expected in-progress state, not a data error.
