# 0001: Account-type-aware, tax-optimal rebalancing

**Status:** Accepted (decision 4 revised 2026-06-04 — see Revision)
**Date:** 2026-06-03

## Context

The rebalancing engine applied the global 5-group target percentage uniformly to
every account's AUM — each account was a clone of the global mix. This is wrong
for Korean tax-advantaged accounts in two independent ways.

**1. Eligibility (a correctness bug, independent of any tax opinion).** Korean
rules constrain what each account type may hold:

| Account type | May hold |
|---|---|
| 위탁 (brokerage) | anything |
| IRP / 연금저축 | domestic-listed ETFs/funds only — **no individual stocks, no foreign-listed securities** |
| ISA (중개형) | domestic-listed only (ETF *or* individual stock) — no foreign-listed |

The engine had no `account_type` and no ETF-vs-stock tag, so its portfolio
fallback could recommend buying a US-listed ETF (e.g. SCHD) into an IRP — an
impossible order. The live data is currently compliant only because holdings
were curated by hand; the bug was latent.

**2. Tax location (the user's original question).** Mirroring the same mix into
every account is the opposite of tax-efficient placement. Verified rules
(2026-06):

- **국내배당** → IRP/ISA strongly preferred (IRP: 과세이연 → 연금소득세 3.3–5.5%
  on withdrawal; ISA: 비과세 200만/서민형 400만 then 9.9% 분리과세, vs 15.4% +
  종합과세 risk in a brokerage account).
- **해외배당** → lean brokerage. The 2025 선환급 폐지 means foreign-dividend
  income is now withheld at 15% at source even inside IRP/ISA, erasing most of
  the shelter; only capital-gain deferral survives.
- **해외성장 / 해외안정** (capital-gain-heavy, domestic-listed ETF) → IRP/ISA
  still good (capital-gain deferral unaffected by the 2025 change).
- **국내성장** → lean brokerage (domestic listed-stock capital gains are largely
  untaxed for retail, so little benefit to occupying scarce tax-advantaged space).
- US-listed ETFs (VYM/SCHD/QQQ/VOO/SPY) → brokerage only (ineligible elsewhere).

**Structural constraints that make this different from the old band logic.**
Contribution caps (ISA ~2,000만/yr, 1억 total; IRP+연금 ~1,800만/yr) and the IRP
55-year withdrawal lock mean assets barely move between accounts. The engine
already isolates cash per account (`TestBuildPlanCashIsolationInvariant`): sells
fund buys *within the same account*, never across. So tax-location is achieved by
**internal swaps** (sell over-target group → buy under-target group in the same
account), and the caps/locks cannot be violated by the engine — they constrain
how the *user* funds accounts, upstream of the cash already present.

**Migration constraint.** `db.Open()` runs the embedded `schema.sql` as
`CREATE TABLE IF NOT EXISTS` only — there is no ALTER and no migration framework,
and the production DB column order (Peewee-authored) must be mirrored because
sqlc reads `SELECT *` in schema order.

## Decision

1. **Add `accounts.account_type`** (`brokerage` / `irp` / `pension` / `isa`;
   NULL = unclassified) and **`stocks.asset_class`** (`etf` / `stock`; NULL =
   unclassified). Both nullable, declared last in `schema.sql`.

2. **Migration** via an idempotent, PRAGMA-guarded `migrate()` in `internal/db`
   run from `Open()` and `OpenMemory()` after the schema exec: `PRAGMA
   table_info` → `ALTER TABLE ADD COLUMN` when missing. Fresh DB → schema creates
   the column → ALTER skipped. Existing DB → CREATE no-op → ALTER fires. One
   mechanism, both paths.

3. **Eligibility predicate `canHold(accountType, ticker, isETF)`** keyed off
   listing (6-digit = domestic) + asset class, never the 국내/해외 group label.
   It guards **buys only** (selling an ineligible holding is always allowed) and
   is ANDed with the existing `RestrictOverseas` toggle. NULL/unknown type →
   blocked (strict): an unclassified account gets no buys until it is classified.

4. **Per-account targets replace the uniform target.** A pure planner
   (`planTargetsByAccountGroup`) emits a per-account, per-group target *value*;
   `buildAccountGroupState` consumes it instead of the global percentage. The
   downstream half-rule sell, buy loop, and cash-isolation invariant are
   unchanged. Allocation is done at the **(group × account type)** level by a
   deterministic greedy over a `_placementScore` matrix, then split across the
   accounts of a type **in proportion to AUM**. Consequences:
   - A single account type (e.g. all brokerage) reproduces the old uniform mirror
     exactly — divergence appears only across *different* types.
   - **nil / unrecognized account_type routes to the uniform global target, not
     zero** — otherwise a strict-`canHold` mask would zero every target and the
     engine would recommend liquidating the whole account. This is a live risk:
     the new columns ship NULL, so all accounts are unclassified until the user
     sets them.

5. **Caps/locks and the IRP 70% 위험자산 cap are NOT enforced in v1** — modeled
   as constants/notes. Cash isolation makes in-engine enforcement of caps/locks
   unnecessary. Note: the greedy will likely produce an all-equity-ETF IRP
   target, which is exactly the case the 70% risk-asset cap would constrain — v1
   relies on the account using 디폴트옵션 (which permits 100% risk assets) or a
   manual check. On record, not silently ignored.

6. **The planner does NOT enforce per-security eligibility** — that is the buy
   guard's job (decision 3). An infeasible target surfaces as an unmet group,
   never an illegal buy. Planner-level eligibility masking is deferred (see
   below).

## Consequences

**Easier**
- The engine can never recommend an impossible buy (US ETF into IRP); it is
  enforced by `canHold` and covered by a regression test.
- Tax logic is centralized in one adjustable constants table (`_placementScore`)
  and one pure function — auditable and testable in isolation.
- Per-account targets are pluggable; the old behavior is the homogeneous-type
  special case, so existing tests stayed green.

**Harder / accepted**
- Under fixed AUMs + eligibility the global target can be infeasible → best-effort
  placement; shortfalls surface via the existing unmet-group reporting.
- `_placementScore` encodes a tax opinion that must be revisited as law changes.
  Least-certain cells, flagged for future audit: **해외배당 → brokerage** (rests
  on the 2025 선환급 폐지) and **국내성장** (domestic capital-gains treatment).
- `account_type` / `asset_class` start NULL and need a one-time UI backfill;
  until then, strict mode blocks buys in unclassified accounts (sells/diagnostics
  still work, and the planner keeps them on the uniform target so they are not
  liquidated).
- **Deferred — planner-level eligibility masking.** The planner is permissive
  (every group × type cell is fillable); the buy guard is the backstop. If a
  group's only available instrument is foreign-listed, an IRP/ISA target for it
  cannot be filled and may cause sell-to-idle-cash churn. Not a concern for the
  current data (every 해외 group has a domestic-listed ETF), but worth adding
  snapshot-aware masking later.

## Revision (2026-06-04) — decision 4 superseded

Decision 4 (per-account, tax-concentrated targets via a greedy over
`_placementScore`, `planTargetsByAccountGroup`) drove **every account** toward its
tax-optimal concentration regardless of whether the portfolio was already
balanced. Two problems surfaced in use:

1. **Over-trading.** An already in-band portfolio still churned all accounts to
   reach per-account concentration — that is a relocation engine, not a
   rebalancer.
2. **Tax-realizing relocation.** Concentrating groups meant selling holdings in
   taxable 위탁 accounts purely to relocate, realizing 양도세/배당소득세 with no
   rebalancing need — the opposite of tax-optimal, and unmodelable precisely
   because holdings carry no cost basis.

**New model — aggregate-band-gated, tax-DIRECTED (one-account-type case
unchanged, but the objective is inverted):**

- Targets and bands hold at the **aggregate (all-accounts) level only**;
  per-account group balance is not a goal. Trade only groups whose portfolio-wide
  weight breaches its band — sell over-band down to target, buy under-band up to
  target. A balanced portfolio produces **zero trades**
  (`TestBuildPlanNoTradesWhenAggregateInBand`).
- `_placementScore` no longer forces concentration; it only **directs** the trades
  a breach already requires: sells are taken from tax-advantaged accounts first
  (no realized gains tax) then from where the group is least tax-appropriate;
  buys fill (account, under-group) cells in descending score so each under-band
  group lands in its tax-home account. Tax-optimal placement is thus reached
  **gradually**, riding necessary trades, never via a one-time tax-realizing move.
- Cash isolation, `canHold` eligibility, nil-account-not-liquidated, and
  unmet/unused reporting are all preserved.

Implemented in `internal/services/rebalance_service.go`:
`buildGroupAggregates`, `computeGroupNetActions`, `allocateSells`, rewritten
`buildBuyRecs`. `planTargetsByAccountGroup` / `buildAccountGroupState` /
`calcSellAmounts` removed. The 해외배당 placement row was also corrected
(ISA > 연금·IRP > 위탁; the 2025 선환급 폐지 does not make taxable 위탁 preferable).

## Revision (2026-06-04) — buy trigger widened to below-target

The 2026-06-04 revision (above) gated buys on **lower-band** breach. This
created an asymmetric trigger: a lone over-band group triggered a sell (proceeds
→ target), but if no other group was below its band, `computeGroupNetActions`
produced zero `buyNeed` — the forced sell proceeds stranded as "잔여 현금(이월)"
with no reinvestment path. User policy has no cash-holding rule; the carry-forward
was an engine artifact, not an intent.

**New rule:** sells remain gated on upper-band breach (unchanged). Buys are now
gated on **below-target** (not below-band). Every group that is not
upper-breached and is below its target absorbs available cash (sell proceeds +
pre-existing idle cash) up to target. Conservation guarantees forced sell
proceeds fit: the sold group reaches target (buyNeed = 0) so it is never
immediately re-bought; the remaining below-target groups absorb the freed cash.

A portfolio exactly at target with zero idle cash still produces zero trades.
Cash isolation, `canHold` eligibility, and unmet/unused reporting are all
preserved — if an account cannot hold any eligible instrument for an under-target
group, the residual cash is still correctly reported as 이월 (legitimate, not a
bug).

Change: `computeGroupNetActions` in `internal/services/rebalance_service.go` —
`switch case a.isLowerBreached` → `if !a.isUpperBreached && below target`.
Tests: `TestBuildPlanReinvestsSellProceedsIntoBelowTargetGroups`,
`TestBuildPlanDeploysPreexistingIdleCash` (new, were RED before fix).

## References
- Phase 1 + 2 implementation: `internal/db/db.go`, `internal/db/schema.sql`,
  `internal/services/rebalance_service.go` (`canHold`, `_placementScore`;
  aggregate engine: `buildGroupAggregates`, `computeGroupNetActions`,
  `allocateSells`, `buildBuyRecs`), `internal/repositories/*`, account/stock edit UI.
- Eligibility & placement research is summarized in the Context section above
  (Korean tax rules as of 2026-06). Load-bearing sources:
  - IRP/연금 hold no individual stocks (ETFs/funds only):
    [KB](https://kbthink.com/year-end-tax/pension-savings-vs-irp.html),
    [Mirae Asset](https://magazine.securities.miraeasset.com/contents.php?idx=156)
  - 2025 외국납부세액 선환급 폐지 → foreign-dividend shelter mostly gone inside
    ISA/연금 (drives 해외배당 → brokerage):
    [Hankyung](https://www.hankyung.com/article/2025020430051),
    [Toss Bank](https://www.tossbank.com/articles/dividendtaxation),
    [KB AM](https://www.kbam.co.kr/board/view/667)
  - 국내배당 brokerage 15.4% + 2,000만 종합과세; ISA 비과세 한도 then 9.9%
    분리과세: [KB](https://kbthink.com/main/asset-management/wealth-manage-tip/kbthink-original/202410/kr-stocktax.html),
    [FSC](https://www.fsc.go.kr/po020201/27339)
  - Could not fully verify (flagged): 슈퍼 ISA expansion (proposed, not passed as
    of 2026-06) and the 크레딧 제도 exact effective dates — excluded from the
    scoring assumptions.
