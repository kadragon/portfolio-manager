#!/usr/bin/env python3
"""Tests for check_cash_availability.py — run with: python3 -m unittest discover -s . -p 'test_*.py'"""

import json
import subprocess
import sys
import unittest
from pathlib import Path

SCRIPT = Path(__file__).parent / "check_cash_availability.py"


def run(payload):
    proc = subprocess.run(
        [sys.executable, str(SCRIPT)], input=json.dumps(payload), capture_output=True, text=True
    )
    return proc.returncode, proc.stdout, proc.stderr


class CheckCashAvailabilityTest(unittest.TestCase):
    def test_buys_fit_within_cash_plus_proceeds_passes(self):
        code, out, _ = run({"accounts": [
            {"name": "ISA", "existing_cash_krw": 1_000_000,
             "sell_proceeds_krw": 500_000, "planned_buys_krw": 1_400_000},
        ]})
        self.assertEqual(code, 0)
        self.assertIn("PASS", out)

    def test_buys_exactly_at_boundary_passes(self):
        code, out, _ = run({"accounts": [
            {"name": "ISA", "existing_cash_krw": 1_000_000,
             "sell_proceeds_krw": 500_000, "planned_buys_krw": 1_500_000},
        ]})
        self.assertEqual(code, 0)
        self.assertIn("PASS", out)

    def test_shortfall_fails_with_named_account_and_amount(self):
        code, out, _ = run({"accounts": [
            {"name": "TOSS", "existing_cash_krw": 100_000,
             "sell_proceeds_krw": 0, "planned_buys_krw": 300_000},
        ]})
        self.assertEqual(code, 1)
        self.assertIn("FAIL", out)
        self.assertIn("[account:TOSS]", out)
        self.assertIn("short by 200000", out)

    def test_account_with_no_sells_reports_off_existing_cash_alone(self):
        code, out, _ = run({"accounts": [
            {"name": "연금저축", "existing_cash_krw": 50_000,
             "sell_proceeds_krw": 0, "planned_buys_krw": 50_000},
        ]})
        self.assertEqual(code, 0)
        self.assertIn("PASS", out)

    def test_missing_accounts_array_exits_nonzero(self):
        code, _, _ = run({})
        self.assertNotEqual(code, 0)


if __name__ == "__main__":
    unittest.main()
