#!/usr/bin/env python3
"""Tests for snapshot.py — run with: python3 -m unittest discover -s . -p 'test_*.py'"""

import json
import sqlite3
import subprocess
import sys
import tempfile
import unittest
from datetime import date, timedelta
from pathlib import Path

SCRIPT = Path(__file__).parent / "snapshot.py"

SCHEMA = """
CREATE TABLE groups (
    id TEXT PRIMARY KEY, name TEXT NOT NULL, target_percentage REAL NOT NULL,
    created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL
);
CREATE TABLE stocks (
    id TEXT PRIMARY KEY, ticker TEXT NOT NULL, group_id TEXT NOT NULL, exchange TEXT,
    created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL, name TEXT NOT NULL,
    asset_class TEXT, security_group TEXT
);
CREATE TABLE accounts (
    id TEXT PRIMARY KEY, name TEXT NOT NULL, cash_balance DECIMAL NOT NULL,
    created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL,
    kis_account_no TEXT, kis_api_key_id INTEGER, account_type TEXT, toss_account_seq INTEGER,
    cash_balance_krw DECIMAL, cash_balance_usd DECIMAL
);
CREATE TABLE holdings (
    id TEXT PRIMARY KEY, account_id TEXT NOT NULL, stock_id TEXT NOT NULL,
    quantity DECIMAL NOT NULL, created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL
);
CREATE TABLE stock_prices (
    id TEXT PRIMARY KEY, ticker TEXT NOT NULL, price DECIMAL NOT NULL, currency TEXT NOT NULL,
    name TEXT NOT NULL, exchange TEXT, price_date DATE NOT NULL,
    created_at DATETIME NOT NULL, updated_at DATETIME NOT NULL
);
"""


def build_db(path, rows):
    con = sqlite3.connect(path)
    con.executescript(SCHEMA)
    now = "2026-01-01T00:00:00"
    for g in rows.get("groups", []):
        con.execute("INSERT INTO groups VALUES (?,?,?,?,?)", (g["id"], g["name"], g["target"], now, now))
    for a in rows.get("accounts", []):
        con.execute(
            "INSERT INTO accounts VALUES (?,?,?,?,?,NULL,NULL,NULL,NULL,NULL,NULL)",
            (a["id"], a["name"], a["cash"], now, now),
        )
    for s in rows.get("stocks", []):
        con.execute(
            "INSERT INTO stocks VALUES (?,?,?,?,?,?,?,NULL,NULL)",
            (s["id"], s["ticker"], s["group_id"], s.get("exchange"), now, now, s["name"]),
        )
    for h in rows.get("holdings", []):
        con.execute(
            "INSERT INTO holdings VALUES (?,?,?,?,?,?)",
            (h["id"], h["account_id"], h["stock_id"], h["qty"], now, now),
        )
    for p in rows.get("prices", []):
        con.execute(
            "INSERT INTO stock_prices VALUES (?,?,?,?,?,?,?,?,?)",
            (p["id"], p["ticker"], p["price"], p["currency"], p["ticker"], p.get("exchange"),
             p["date"], now, now),
        )
    con.commit()
    con.close()


def run_snapshot(db_path, fx="1400"):
    proc = subprocess.run(
        [sys.executable, str(SCRIPT), "--db", db_path, "--fx", fx],
        capture_output=True, text=True,
    )
    return proc.returncode, proc.stdout, proc.stderr


class SnapshotTest(unittest.TestCase):
    def test_missing_db_file_errors_with_friendly_message(self):
        code, out, err = run_snapshot("/tmp/does-not-exist-portfolio-xyz.db")
        self.assertNotEqual(code, 0)
        self.assertIn("db not found", err)

    def test_basic_snapshot_computes_weights_including_cash(self):
        with tempfile.TemporaryDirectory() as d:
            db_path = f"{d}/portfolio.db"
            today = date.today().isoformat()
            build_db(db_path, {
                "groups": [{"id": "g1", "name": "국내성장", "target": 50}],
                "accounts": [{"id": "a1", "name": "ISA", "cash": 1_000_000}],
                "stocks": [{"id": "s1", "ticker": "005930", "group_id": "g1", "name": "삼성전자"}],
                "holdings": [{"id": "h1", "account_id": "a1", "stock_id": "s1", "qty": 10}],
                "prices": [{"id": "p1", "ticker": "005930", "price": 90_000,
                            "currency": "KRW", "date": today}],
            })
            code, out, _ = run_snapshot(db_path)
            self.assertEqual(code, 0)
            data = json.loads(out)
            self.assertEqual(data["total_holdings_krw"], 900_000)
            self.assertEqual(data["total_cash_krw"], 1_000_000)
            self.assertEqual(data["total_krw"], 1_900_000)
            group = next(g for g in data["groups"] if g["name"] == "국내성장")
            self.assertAlmostEqual(group["weight_pct"], 900_000 / 1_900_000 * 100, places=2)
            self.assertEqual(data["warnings"], [])

    def test_target_only_group_with_no_holdings_still_appears_at_zero(self):
        with tempfile.TemporaryDirectory() as d:
            db_path = f"{d}/portfolio.db"
            build_db(db_path, {
                "groups": [{"id": "g1", "name": "금", "target": 10}],
                "accounts": [{"id": "a1", "name": "ISA", "cash": 100_000}],
            })
            code, out, _ = run_snapshot(db_path)
            self.assertEqual(code, 0)
            data = json.loads(out)
            group = next(g for g in data["groups"] if g["name"] == "금")
            self.assertEqual(group["value_krw"], 0)
            self.assertEqual(group["db_target_pct"], 10)

    def test_missing_price_warns_and_sets_currency_null_not_krw(self):
        with tempfile.TemporaryDirectory() as d:
            db_path = f"{d}/portfolio.db"
            build_db(db_path, {
                "groups": [{"id": "g1", "name": "국내성장", "target": 50}],
                "accounts": [{"id": "a1", "name": "ISA", "cash": 0}],
                "stocks": [{"id": "s1", "ticker": "999999", "group_id": "g1", "name": "신규종목"}],
                "holdings": [{"id": "h1", "account_id": "a1", "stock_id": "s1", "qty": 5}],
            })
            code, out, _ = run_snapshot(db_path)
            self.assertEqual(code, 0)
            data = json.loads(out)
            self.assertTrue(any("no price for 999999" in w for w in data["warnings"]))
            holding = data["accounts"][0]["holdings"][0]
            self.assertEqual(holding["value_krw"], 0)
            self.assertIsNone(holding["currency"])

    def test_non_positive_price_warns_and_zeroes_value(self):
        with tempfile.TemporaryDirectory() as d:
            db_path = f"{d}/portfolio.db"
            today = date.today().isoformat()
            build_db(db_path, {
                "groups": [{"id": "g1", "name": "국내성장", "target": 50}],
                "accounts": [{"id": "a1", "name": "ISA", "cash": 0}],
                "stocks": [{"id": "s1", "ticker": "005930", "group_id": "g1", "name": "삼성전자"}],
                "holdings": [{"id": "h1", "account_id": "a1", "stock_id": "s1", "qty": 5}],
                "prices": [{"id": "p1", "ticker": "005930", "price": 0,
                            "currency": "KRW", "date": today}],
            })
            code, out, _ = run_snapshot(db_path)
            self.assertEqual(code, 0)
            data = json.loads(out)
            self.assertTrue(any("non-positive price" in w for w in data["warnings"]))
            self.assertEqual(data["accounts"][0]["holdings"][0]["value_krw"], 0)

    def test_stale_price_warns_past_stale_days_threshold(self):
        with tempfile.TemporaryDirectory() as d:
            db_path = f"{d}/portfolio.db"
            old_date = (date.today() - timedelta(days=30)).isoformat()
            build_db(db_path, {
                "groups": [{"id": "g1", "name": "국내성장", "target": 50}],
                "accounts": [{"id": "a1", "name": "ISA", "cash": 0}],
                "stocks": [{"id": "s1", "ticker": "005930", "group_id": "g1", "name": "삼성전자"}],
                "holdings": [{"id": "h1", "account_id": "a1", "stock_id": "s1", "qty": 5}],
                "prices": [{"id": "p1", "ticker": "005930", "price": 90_000,
                            "currency": "KRW", "date": old_date}],
            })
            code, out, _ = run_snapshot(db_path)
            self.assertEqual(code, 0)
            data = json.loads(out)
            self.assertTrue(any("days old" in w for w in data["warnings"]))

    def test_usd_holding_converted_by_fx_rate(self):
        with tempfile.TemporaryDirectory() as d:
            db_path = f"{d}/portfolio.db"
            today = date.today().isoformat()
            build_db(db_path, {
                "groups": [{"id": "g1", "name": "TOSS", "target": 50}],
                "accounts": [{"id": "a1", "name": "TOSS", "cash": 0}],
                "stocks": [{"id": "s1", "ticker": "AAPL", "group_id": "g1", "name": "Apple"}],
                "holdings": [{"id": "h1", "account_id": "a1", "stock_id": "s1", "qty": 2}],
                "prices": [{"id": "p1", "ticker": "AAPL", "price": 100,
                            "currency": "USD", "date": today}],
            })
            code, out, _ = run_snapshot(db_path, fx="1400")
            self.assertEqual(code, 0)
            data = json.loads(out)
            self.assertEqual(data["accounts"][0]["holdings"][0]["value_krw"], 280_000)


if __name__ == "__main__":
    unittest.main()
