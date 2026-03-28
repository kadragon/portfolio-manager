#!/usr/bin/env python3
"""One-time migration script: Supabase → SQLite.

Usage:
    # Ensure .env has SUPABASE_URL and SUPABASE_SERVICE_ROLE_KEY
    .venv/bin/python scripts/migrate_supabase_to_sqlite.py

This reads all data from Supabase and writes it to .data/portfolio.db.
Run BEFORE removing the supabase dependency from pyproject.toml.
"""

import json
import os
import sys
from datetime import datetime
from pathlib import Path
from uuid import UUID

from dotenv import load_dotenv

# Add src to path
sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

load_dotenv()


def main():
    # Import supabase (still available before removal)
    try:
        from supabase import Client, create_client  # type: ignore[reportMissingImports]
    except ImportError:
        print("ERROR: supabase package not installed. Run this before removing it.")
        sys.exit(1)

    url = os.getenv("SUPABASE_URL")
    key = os.getenv("SUPABASE_SERVICE_ROLE_KEY")
    if not url or not key:
        print("ERROR: SUPABASE_URL and SUPABASE_SERVICE_ROLE_KEY must be set in .env")
        sys.exit(1)

    print("Connecting to Supabase...")
    client: Client = create_client(url, key)

    # Import Peewee models
    from portfolio_manager.services.database import (
        AccountModel,
        DepositModel,
        GroupModel,
        HoldingModel,
        OrderExecutionModel,
        StockModel,
        StockPriceModel,
        init_db,
    )

    db_path = ".data/portfolio.db"
    if os.path.exists(db_path):
        print(f"WARNING: {db_path} already exists. Will be overwritten.")
        os.remove(db_path)

    print(f"Initializing SQLite at {db_path}...")
    init_db(db_path)

    # Table migration order (respects foreign keys)
    tables = [
        ("groups", GroupModel, _migrate_group),
        ("stocks", StockModel, _migrate_stock),
        ("accounts", AccountModel, _migrate_account),
        ("holdings", HoldingModel, _migrate_holding),
        ("deposits", DepositModel, _migrate_deposit),
        ("stock_prices", StockPriceModel, _migrate_stock_price),
        ("order_executions", OrderExecutionModel, _migrate_order_execution),
    ]

    for table_name, model, migrate_fn in tables:
        print(f"  Migrating {table_name}...", end=" ")
        response = client.table(table_name).select("*").execute()
        rows = response.data or []

        if rows:
            with model._meta.database.atomic():
                for row in rows:
                    migrate_fn(model, row)

        print(f"{len(rows)} rows")

    print("\nDone! Verifying row counts...")
    for table_name, model, _ in tables:
        count = model.select().count()
        print(f"  {table_name}: {count}")

    print(f"\nSQLite database saved to: {db_path}")


def _parse_dt(val):
    if val is None:
        return datetime.now()
    return datetime.fromisoformat(str(val))


def _migrate_group(model, row):
    model.create(
        id=UUID(row["id"]),
        name=row["name"],
        target_percentage=float(row.get("target_percentage", 0.0)),
        created_at=_parse_dt(row.get("created_at")),
        updated_at=_parse_dt(row.get("updated_at")),
    )


def _migrate_stock(model, row):
    model.create(
        id=UUID(row["id"]),
        ticker=row["ticker"],
        group=UUID(row["group_id"]),
        exchange=row.get("exchange"),
        created_at=_parse_dt(row.get("created_at")),
        updated_at=_parse_dt(row.get("updated_at")),
    )


def _migrate_account(model, row):
    model.create(
        id=UUID(row["id"]),
        name=row["name"],
        cash_balance=row["cash_balance"],
        created_at=_parse_dt(row.get("created_at")),
        updated_at=_parse_dt(row.get("updated_at")),
    )


def _migrate_holding(model, row):
    model.create(
        id=UUID(row["id"]),
        account=UUID(row["account_id"]),
        stock=UUID(row["stock_id"]),
        quantity=row["quantity"],
        created_at=_parse_dt(row.get("created_at")),
        updated_at=_parse_dt(row.get("updated_at")),
    )


def _migrate_deposit(model, row):
    model.create(
        id=UUID(row["id"]),
        amount=row["amount"],
        deposit_date=row["deposit_date"],
        note=row.get("note"),
        created_at=_parse_dt(row.get("created_at")),
        updated_at=_parse_dt(row.get("updated_at")),
    )


def _migrate_stock_price(model, row):
    model.create(
        id=UUID(row["id"]),
        ticker=row["ticker"],
        price=row["price"],
        currency=row["currency"],
        name=row.get("name", ""),
        exchange=row.get("exchange"),
        price_date=row["price_date"],
        created_at=_parse_dt(row.get("created_at")),
        updated_at=_parse_dt(row.get("updated_at")),
    )


def _migrate_order_execution(model, row):
    raw = row.get("raw_response")
    if isinstance(raw, dict):
        raw = json.dumps(raw)
    model.create(
        id=UUID(row["id"]),
        ticker=row["ticker"],
        side=row["side"],
        quantity=row["quantity"],
        currency=row["currency"],
        exchange=row.get("exchange"),
        status=row["status"],
        message=row.get("message", ""),
        raw_response=raw,
        created_at=_parse_dt(row.get("created_at")),
    )


if __name__ == "__main__":
    main()
