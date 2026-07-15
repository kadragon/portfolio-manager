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

    def test_cash_overrun_in_one_account_does_not_spuriously_fail_unrelated_group(self):
        # Regression: a negative remainder from a cash-overrun account used to be summed
        # into post_cash_total, deflating the denominator for every group's post_pct check
        # and manufacturing unrelated band failures. It must now report only the cash
        # failure, not a downstream group-band failure for a group whose trade is fine.
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
        account_failures = [l for l in out.splitlines() if "[account:" in l]
        group_failures = [l for l in out.splitlines() if "[group:" in l]
        self.assertEqual(len(account_failures), 1)
        self.assertEqual(len(group_failures), 0,
                          f"expected no spurious group failure, got: {group_failures}")

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
