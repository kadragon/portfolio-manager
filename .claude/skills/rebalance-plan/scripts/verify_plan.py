#!/usr/bin/env python3
"""Decimal-based invariant check for a rebalance-plan trade plan.

Float arithmetic on won-scale sums silently drifts; this replaces the
inline ad-hoc check with a fixed, reusable one. Exit 0 = all invariants
hold; exit 1 = at least one failed (see printed detail).

Input: JSON on stdin or --file, shaped as:
{
  "accounts": [
    {"name": "ISA", "pre_value_krw": 50000000, "cash_krw": 1000000,
     "buys_krw": 2000000, "sells_krw": 1500000}
  ],
  "groups": [
    {"name": "금", "pre_value_krw": 12000000, "trade_krw": 3000000,
     "effective_target_pct": 20, "band_pct": 5}
  ]
}

trade_krw is net (buy positive, sell negative) at the group level.
effective_target_pct / band_pct: use the interim target and its band if
the group is mid §Transition Schedule — see SKILL.md step 2.
"""

import argparse
import json
import sys
from decimal import Decimal, InvalidOperation


def D(v):
    try:
        return Decimal(str(v))
    except InvalidOperation:
        raise SystemExit(f"bad decimal value: {v!r}")


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--file", help="JSON file (default: stdin)")
    ap.add_argument("--remainder-pct", type=float, default=1.0,
                     help="max allowed unused cash remainder, %% of account value")
    args = ap.parse_args()

    raw = open(args.file).read() if args.file else sys.stdin.read()
    data = json.loads(raw)

    failures = []

    # 1. per-account cash constraint
    for a in data.get("accounts", []):
        cash, buys, sells = D(a["cash_krw"]), D(a["buys_krw"]), D(a["sells_krw"])
        pre_value = D(a["pre_value_krw"])
        available = sells + cash
        if buys > available:
            failures.append(
                f"[account:{a['name']}] buys {buys} exceed sells+cash {available}"
            )
        remainder = available - buys
        limit = pre_value * D(args.remainder_pct) / D(100)
        if pre_value > 0 and remainder > limit:
            failures.append(
                f"[account:{a['name']}] unused remainder {remainder} exceeds "
                f"{args.remainder_pct}% of account value ({limit}) — explain or fix"
            )

    # 2. post-trade group weights within band of effective target
    groups = data.get("groups", [])
    pre_total = sum(D(g["pre_value_krw"]) for g in groups)
    trade_total = sum(D(g["trade_krw"]) for g in groups)
    post_total = pre_total + trade_total
    if post_total > 0:
        for g in groups:
            post_value = D(g["pre_value_krw"]) + D(g["trade_krw"])
            post_pct = post_value / post_total * D(100)
            target = D(g["effective_target_pct"])
            band = D(g["band_pct"])
            dev = abs(post_pct - target)
            if dev > band:
                failures.append(
                    f"[group:{g['name']}] post-trade {post_pct:.2f}% is "
                    f"{dev:.2f}pp from target {target}% (band {band}pp) — explain or fix"
                )

    # 3. no money invented: sum of account deltas == sum of group deltas
    account_delta = sum(D(a["buys_krw"]) - D(a["sells_krw"]) for a in data.get("accounts", []))
    if account_delta != trade_total:
        failures.append(
            f"account-level net trade {account_delta} != group-level net trade {trade_total} "
            "— a trade was double-counted or dropped"
        )

    if failures:
        print("FAIL")
        for f in failures:
            print(f" - {f}")
        sys.exit(1)
    print("PASS — all invariants hold")


if __name__ == "__main__":
    main()
