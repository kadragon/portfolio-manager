#!/usr/bin/env python3
"""Deterministic band/deviation check for rebalance-plan step 2.

Takes each group's current weight (from snapshot.py) and its effective
target/band (read from the policy file by the agent — this script does not
parse policy semantics) and reports which groups need a trade. Replaces
hand arithmetic, which is exactly where a KRW-scale float slip turns a
true in-band group into a false breach or vice versa.

Input: JSON on stdin or --file, shaped as:
{
  "groups": [
    {"name": "국내성장", "weight_pct": 32.1, "effective_target_pct": 25,
     "band_pct": 5, "hard_ceiling_pct": 30}
  ]
}

`hard_ceiling_pct` is optional — set it only for a group with a mechanical
override rule (e.g. the KOSPI exit rule, 국내성장 >= 30%). Omit it for every
other group; a hard-ceiling breach is reported independently of the band.

Placement violations (an asset sitting in an account the policy forbids) are
not computed here — they need policy semantics, not group-level numbers.
Pass each one via --placement-violation (repeatable) so it appears in the
same report as a forced trade.

Always exits 0 — this is a report, not a pass/fail gate (in contrast to
verify_plan.py, which does gate). Exits 1 only on malformed input.
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
    ap.add_argument("--placement-violation", action="append", default=[],
                     dest="placement_violations",
                     help="description of a forced trade unrelated to group deviation; repeatable")
    args = ap.parse_args()

    raw = open(args.file).read() if args.file else sys.stdin.read()
    data = json.loads(raw)

    groups = data.get("groups")
    if not groups:
        raise SystemExit("input missing non-empty 'groups' array")

    any_trade = False
    for g in groups:
        weight = D(g["weight_pct"])
        target = D(g["effective_target_pct"])
        band = D(g["band_pct"])
        dev = weight - target
        in_band = abs(dev) <= band

        hard_ceiling = g.get("hard_ceiling_pct")
        hard_breach = hard_ceiling is not None and weight >= D(hard_ceiling)

        if hard_breach:
            any_trade = True
            print(f"TRADE-NEEDED [{g['name']}] hard ceiling breach: "
                  f"weight {weight}% >= {hard_ceiling}%")
        elif not in_band:
            any_trade = True
            print(f"TRADE-NEEDED [{g['name']}] band breach: "
                  f"{weight}% is {abs(dev):.2f}pp from target {target}% (band {band}pp)")
        else:
            print(f"NO-TRADE     [{g['name']}] {weight}% within {band}pp of target {target}%")

    for v in args.placement_violations:
        any_trade = True
        print(f"TRADE-NEEDED [placement violation] {v}")

    print()
    print("some groups need trades" if any_trade else "no groups need trades — 이번 점검 매매 불필요")


if __name__ == "__main__":
    main()
