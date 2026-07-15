#!/usr/bin/env python3
"""Decimal-based Phase-B cash availability check for execute-rebalance-plan.

Sells finish before buys; before Phase B starts, each account's still-to-run
buys must not exceed what Phase A actually delivered (successful sells only)
plus existing cash. Replaces hand arithmetic on real-money trade sequencing —
exactly where a slip is costliest.

Input: JSON on stdin or --file, shaped as:
{
  "accounts": [
    {"name": "ISA", "existing_cash_krw": 1000000,
     "sell_proceeds_krw": 500000, "planned_buys_krw": 1400000}
  ]
}

sell_proceeds_krw: sum of only the sells that actually succeeded this run
(skip failed/skipped lines). planned_buys_krw: sum of the buy lines still to
execute in Phase B for that account.

Exit 0 = every account's buys fit; exit 1 = at least one account is short
(see printed detail) — this gates Phase B, it does not decide what to do
about a shortfall (that's the agent's job: ask the user).
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
    args = ap.parse_args()

    raw = open(args.file).read() if args.file else sys.stdin.read()
    data = json.loads(raw)

    accounts = data.get("accounts")
    if not accounts:
        raise SystemExit("input missing non-empty 'accounts' array")

    failures = []
    for a in accounts:
        cash = D(a["existing_cash_krw"])
        proceeds = D(a["sell_proceeds_krw"])
        buys = D(a["planned_buys_krw"])
        available = cash + proceeds
        if buys > available:
            shortfall = buys - available
            failures.append(
                f"[account:{a['name']}] planned buys {buys} exceed available "
                f"{available} (cash {cash} + sell proceeds {proceeds}) — short by {shortfall}"
            )

    if failures:
        print("FAIL")
        for f in failures:
            print(f" - {f}")
        sys.exit(1)
    print("PASS — every account's buys fit within existing cash + successful sell proceeds")


if __name__ == "__main__":
    main()
