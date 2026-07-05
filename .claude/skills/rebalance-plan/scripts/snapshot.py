#!/usr/bin/env python3
"""Portfolio snapshot for rebalance planning.

Reads holdings, latest prices, and group targets from the portfolio-manager
SQLite DB and prints a JSON snapshot. Deterministic part of the rebalance-plan
skill — trade planning and tax judgment stay with the agent.

Usage:
    python3 snapshot.py --fx 1400 [--db .data/portfolio.db]
"""

import argparse
import json
import sqlite3
import sys
from datetime import date

LATEST_PRICE_SQL = """
SELECT price, currency, price_date FROM stock_prices
WHERE ticker = ? ORDER BY price_date DESC LIMIT 1
"""


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--db", default=".data/portfolio.db")
    ap.add_argument("--fx", type=float, required=True, help="USD/KRW rate")
    ap.add_argument("--stale-days", type=int, default=7,
                    help="warn when latest price is older than this many days")
    args = ap.parse_args()

    con = sqlite3.connect(f"file:{args.db}?mode=ro", uri=True)
    con.row_factory = sqlite3.Row

    warnings = []
    accounts = {}
    for r in con.execute(
        "SELECT id, name, account_type, cash_balance FROM accounts ORDER BY name"
    ):
        accounts[r["id"]] = {
            "name": r["name"],
            "type": r["account_type"],
            "cash_krw": float(r["cash_balance"]),
            "holdings": [],
        }

    db_targets = {
        r["name"]: r["target_percentage"]
        for r in con.execute("SELECT name, target_percentage FROM groups")
    }
    # Seed every DB group at 0 so target-only groups (no holdings yet, e.g. a
    # newly added 금/채권 allocation) still appear in the output — otherwise the
    # planning agent can't see their underweight at all.
    group_totals = {name: 0.0 for name in db_targets}

    rows = con.execute("""
        SELECT h.account_id, s.ticker, s.name AS stock_name, s.exchange,
               g.name AS grp, h.quantity
        FROM holdings h
        JOIN stocks s ON s.id = h.stock_id
        JOIN groups g ON g.id = s.group_id
    """).fetchall()

    today = date.today()
    for r in rows:
        p = con.execute(LATEST_PRICE_SQL, (r["ticker"],)).fetchone()
        if p is None:
            warnings.append(f"no price for {r['ticker']} — value set to 0")
            price, currency, value = 0.0, "KRW", 0.0
        elif float(p["price"]) <= 0:
            warnings.append(
                f"non-positive price {p['price']} for {r['ticker']} ({p['price_date'][:10]}) "
                "— likely bad data, value set to 0, do not trust this group's weight"
            )
            price, currency, value = 0.0, p["currency"], 0.0
        else:
            price, currency = float(p["price"]), p["currency"]
            age = (today - date.fromisoformat(p["price_date"][:10])).days
            if age > args.stale_days:
                warnings.append(
                    f"price for {r['ticker']} is {age} days old ({p['price_date'][:10]})"
                )
            value = float(r["quantity"]) * price * (args.fx if currency == "USD" else 1.0)
        accounts[r["account_id"]]["holdings"].append({
            "ticker": r["ticker"],
            "name": r["stock_name"],
            "exchange": r["exchange"],
            "group": r["grp"],
            "qty": float(r["quantity"]),
            "price": price,
            "currency": currency,
            "value_krw": round(value),
        })
        group_totals[r["grp"]] = group_totals.get(r["grp"], 0.0) + value

    total_holdings = sum(group_totals.values())
    total_cash = sum(a["cash_krw"] for a in accounts.values())
    total = total_holdings + total_cash

    out = {
        "as_of": today.isoformat(),
        "fx_usdkrw": args.fx,
        "total_krw": round(total),
        "total_holdings_krw": round(total_holdings),
        "total_cash_krw": round(total_cash),
        "accounts": sorted(accounts.values(), key=lambda a: -sum(
            h["value_krw"] for h in a["holdings"])),
        "groups": [
            {
                "name": g,
                "db_target_pct": db_targets.get(g),
                "value_krw": round(v),
                "weight_pct": round(100 * v / total, 2) if total > 0 else None,
            }
            for g, v in sorted(group_totals.items(), key=lambda kv: -kv[1])
        ],
        "warnings": warnings,
    }
    json.dump(out, sys.stdout, ensure_ascii=False, indent=2)
    print()


if __name__ == "__main__":
    main()
