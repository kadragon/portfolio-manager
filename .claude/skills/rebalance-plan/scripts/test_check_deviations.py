#!/usr/bin/env python3
"""Tests for check_deviations.py — run with: python3 -m unittest discover -s . -p 'test_*.py'"""

import json
import subprocess
import sys
import unittest
from pathlib import Path

SCRIPT = Path(__file__).parent / "check_deviations.py"


def run(payload, extra_args=None):
    args = [sys.executable, str(SCRIPT)] + (extra_args or [])
    proc = subprocess.run(
        args, input=json.dumps(payload), capture_output=True, text=True
    )
    return proc.returncode, proc.stdout, proc.stderr


class CheckDeviationsTest(unittest.TestCase):
    def test_in_band_group_reports_no_trade(self):
        code, out, _ = run({"groups": [
            {"name": "국내성장", "weight_pct": 27, "effective_target_pct": 25, "band_pct": 5},
        ]})
        self.assertEqual(code, 0)
        self.assertIn("NO-TRADE     [국내성장]", out)
        self.assertIn("no groups need trades", out)

    def test_band_boundary_is_inclusive_in_band(self):
        # deviation exactly equals band -> still NO-TRADE (abs(dev) <= band)
        code, out, _ = run({"groups": [
            {"name": "채권", "weight_pct": 30, "effective_target_pct": 25, "band_pct": 5},
        ]})
        self.assertEqual(code, 0)
        self.assertIn("NO-TRADE     [채권]", out)

    def test_band_breach_reports_trade_needed(self):
        code, out, _ = run({"groups": [
            {"name": "채권", "weight_pct": 30.01, "effective_target_pct": 25, "band_pct": 5},
        ]})
        self.assertEqual(code, 0)
        self.assertIn("TRADE-NEEDED [채권] band breach", out)
        self.assertIn("some groups need trades", out)

    def test_hard_ceiling_breach_overrides_in_band_reading(self):
        # weight is within band of target but still >= hard ceiling
        code, out, _ = run({"groups": [
            {"name": "국내성장", "weight_pct": 30, "effective_target_pct": 27,
             "band_pct": 5, "hard_ceiling_pct": 30},
        ]})
        self.assertEqual(code, 0)
        self.assertIn("TRADE-NEEDED [국내성장] hard ceiling breach", out)

    def test_hard_ceiling_absent_for_other_groups_does_not_trigger(self):
        code, out, _ = run({"groups": [
            {"name": "금", "weight_pct": 10, "effective_target_pct": 10, "band_pct": 5},
        ]})
        self.assertEqual(code, 0)
        self.assertIn("NO-TRADE     [금]", out)

    def test_placement_violation_forces_trade_needed_independent_of_groups(self):
        code, out, _ = run(
            {"groups": [
                {"name": "채권", "weight_pct": 25, "effective_target_pct": 25, "band_pct": 5},
            ]},
            extra_args=["--placement-violation", "KR-listed overseas ETF in TOSS"],
        )
        self.assertEqual(code, 0)
        self.assertIn("TRADE-NEEDED [placement violation] KR-listed overseas ETF in TOSS", out)
        self.assertIn("some groups need trades", out)

    def test_missing_groups_array_exits_nonzero(self):
        code, _, _ = run({})
        self.assertNotEqual(code, 0)

    def test_bad_decimal_value_exits_nonzero(self):
        code, _, _ = run({"groups": [
            {"name": "금", "weight_pct": "not-a-number", "effective_target_pct": 10, "band_pct": 5},
        ]})
        self.assertNotEqual(code, 0)


if __name__ == "__main__":
    unittest.main()
