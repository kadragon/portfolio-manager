#!/usr/bin/env python3
"""Tests for verify_plan.py — run with: python3 -m unittest discover -s . -p 'test_*.py'"""

import json
import subprocess
import sys
import unittest
from pathlib import Path

SCRIPT = Path(__file__).parent / "verify_plan.py"


def run(payload, extra_args=None):
    args = [sys.executable, str(SCRIPT)] + (extra_args or [])
    proc = subprocess.run(
        args, input=json.dumps(payload), capture_output=True, text=True
    )
    return proc.returncode, proc.stdout, proc.stderr


class VerifyPlanTest(unittest.TestCase):
    def test_balanced_plan_passes(self):
        code, out, _ = run({
            "accounts": [
                {"name": "ISA", "pre_value_krw": 50_000_000, "cash_krw": 1_000_000,
                 "buys_krw": 1_500_000, "sells_krw": 1_000_000},
            ],
            "groups": [
                {"name": "채권", "pre_value_krw": 10_000_000, "trade_krw": 500_000,
                 "effective_target_pct": 20, "band_pct": 5},
                {"name": "국내성장", "pre_value_krw": 40_000_000, "trade_krw": 0,
                 "effective_target_pct": 80, "band_pct": 5},
            ],
        })
        self.assertEqual(code, 0)
        self.assertIn("PASS", out)

    def test_buys_exceeding_available_cash_fails(self):
        code, out, _ = run({
            "accounts": [
                {"name": "ISA", "pre_value_krw": 50_000_000, "cash_krw": 100_000,
                 "buys_krw": 5_000_000, "sells_krw": 0},
            ],
            "groups": [
                {"name": "국내성장", "pre_value_krw": 45_000_000, "trade_krw": 5_000_000,
                 "effective_target_pct": 100, "band_pct": 5},
            ],
        })
        self.assertEqual(code, 1)
        self.assertIn("FAIL", out)
        self.assertIn("buys 5000000 exceed sells+cash 100000", out)

    def test_unrelated_group_post_pct_is_invariant_to_a_different_accounts_cash_overrun(self):
        # post_total for check 2 equals pre_total(groups) + sum(pre-trade cash across all
        # accounts) whenever check 3 (money conservation) holds — the buys/sells terms cancel
        # out algebraically regardless of any single account's shortfall. So an ISA cash
        # overrun must NOT change an unrelated group's (held in TOSS) computed post_pct at
        # all, no matter how large the overrun is. This pins that invariant directly: two
        # different overrun magnitudes in ISA must yield the identical post_pct for 채권.
        import re

        def build(overrun_buys_krw):
            return {
                "accounts": [
                    {"name": "ISA", "pre_value_krw": 50_000_000, "cash_krw": 100_000,
                     "buys_krw": overrun_buys_krw, "sells_krw": 0},
                    {"name": "TOSS", "pre_value_krw": 20_000_000, "cash_krw": 2_000_000,
                     "buys_krw": 0, "sells_krw": 0},
                ],
                "groups": [
                    {"name": "국내성장", "pre_value_krw": 45_000_000, "trade_krw": overrun_buys_krw,
                     "effective_target_pct": 100, "band_pct": 5},
                    # target/band chosen so 채권 always fails, so its computed post_pct is
                    # visible in the FAIL line for both overrun sizes below.
                    {"name": "채권", "pre_value_krw": 20_000_000, "trade_krw": 0,
                     "effective_target_pct": 0, "band_pct": 0},
                ],
            }

        pct_re = re.compile(r"\[group:채권\] post-trade ([\d.]+)% ")
        code_a, out_a, _ = run(build(5_000_000))
        code_b, out_b, _ = run(build(7_500_000))
        self.assertEqual(code_a, 1)
        self.assertEqual(code_b, 1)
        pct_a = pct_re.search(out_a).group(1)
        pct_b = pct_re.search(out_b).group(1)
        self.assertEqual(pct_a, pct_b,
                          f"채권 post_pct should be invariant to ISA's overrun size: {pct_a} != {pct_b}")
        # 20,000,000 / (65,000,000 groups pre_total + 2,100,000 total pre cash) * 100
        self.assertEqual(pct_a, "29.81")

    def test_account_cash_overrun_still_reported_as_account_failure(self):
        code, out, _ = run({
            "accounts": [
                {"name": "ISA", "pre_value_krw": 50_000_000, "cash_krw": 100_000,
                 "buys_krw": 5_000_000, "sells_krw": 0},
            ],
            "groups": [
                {"name": "국내성장", "pre_value_krw": 45_000_000, "trade_krw": 5_000_000,
                 "effective_target_pct": 100, "band_pct": 5},
            ],
        })
        self.assertEqual(code, 1)
        self.assertIn("[account:", out)

    def test_group_post_trade_weight_outside_band_fails(self):
        # One account, buys == available (remainder 0, no remainder-check noise) and
        # account_delta == trade_total (no check-3 noise) — isolates check #2 (group band).
        code, out, _ = run({
            "accounts": [
                {"name": "ISA", "pre_value_krw": 50_000_000, "cash_krw": 1_000_000,
                 "buys_krw": 2_000_000, "sells_krw": 1_000_000},
            ],
            "groups": [
                {"name": "금", "pre_value_krw": 1_000_000, "trade_krw": 2_000_000,
                 "effective_target_pct": 1, "band_pct": 1},
                {"name": "국내성장", "pre_value_krw": 49_000_000, "trade_krw": -1_000_000,
                 "effective_target_pct": 94, "band_pct": 100},
            ],
        })
        self.assertEqual(code, 1)
        self.assertIn("[group:금]", out)
        self.assertNotIn("[account:", out)

    def test_account_group_net_trade_mismatch_fails(self):
        code, out, _ = run({
            "accounts": [
                {"name": "ISA", "pre_value_krw": 50_000_000, "cash_krw": 1_000_000,
                 "buys_krw": 2_000_000, "sells_krw": 1_000_000},
            ],
            "groups": [
                {"name": "국내성장", "pre_value_krw": 49_000_000, "trade_krw": 500_000,
                 "effective_target_pct": 100, "band_pct": 5},
            ],
        })
        self.assertEqual(code, 1)
        self.assertIn("account-level net trade", out)
        self.assertIn("!= group-level net trade", out)

    def test_missing_arrays_exits_nonzero(self):
        code, _, _ = run({"accounts": [], "groups": []})
        self.assertNotEqual(code, 0)


if __name__ == "__main__":
    unittest.main()
