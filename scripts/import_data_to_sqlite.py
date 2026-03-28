#!/usr/bin/env python3
"""Import data from Supabase JSON dumps into SQLite.

Data extracted via Supabase MCP SQL queries.
"""

import os
import sys
from datetime import datetime
from decimal import Decimal
from pathlib import Path
from uuid import UUID

sys.path.insert(0, str(Path(__file__).resolve().parents[1] / "src"))

from portfolio_manager.services.database import (
    AccountModel,
    DepositModel,
    GroupModel,
    HoldingModel,
    StockModel,
    db,
    init_db,
)


def parse_dt(val):
    if val is None:
        return datetime.now()
    return datetime.fromisoformat(str(val))


# ── Data from Supabase MCP SQL queries ──

GROUPS = [
    {
        "id": "3ae6cace-2a8c-44e6-a4d2-ee17c67b07c4",
        "name": "해외배당",
        "target_percentage": 15,
        "created_at": "2026-01-03T14:32:38.655351+00:00",
        "updated_at": "2026-02-01T02:49:13.023841+00:00",
    },
    {
        "id": "a1ae778b-7705-405b-b0b2-31de492e0812",
        "name": "해외성장",
        "target_percentage": 25,
        "created_at": "2026-01-03T14:32:50.533658+00:00",
        "updated_at": "2026-02-05T04:21:18.232727+00:00",
    },
    {
        "id": "2ca48716-5990-4612-8314-e7336c1a2d61",
        "name": "국내배당",
        "target_percentage": 15,
        "created_at": "2026-01-03T14:32:32.730863+00:00",
        "updated_at": "2026-02-12T02:23:15.302316+00:00",
    },
    {
        "id": "6ea05152-a457-4e4d-a86b-0a4278b7f3d5",
        "name": "국내성장",
        "target_percentage": 35,
        "created_at": "2026-01-03T14:32:29.257597+00:00",
        "updated_at": "2026-02-12T02:23:27.228743+00:00",
    },
    {
        "id": "e554b7d1-8cd1-44e1-8d8c-9bc8532c0612",
        "name": "해외안정",
        "target_percentage": 10,
        "created_at": "2026-01-03T14:32:54.958657+00:00",
        "updated_at": "2026-02-12T02:23:54.963495+00:00",
    },
]

STOCKS = [
    {
        "id": "3907f51f-343c-4f01-8d91-36ba26498258",
        "ticker": "310970",
        "group_id": "6ea05152-a457-4e4d-a86b-0a4278b7f3d5",
        "exchange": None,
        "created_at": "2026-01-03T14:35:26.738837+00:00",
        "updated_at": "2026-01-03T14:35:26.738837+00:00",
    },
    {
        "id": "2b39913c-402d-4bac-97c3-2fb3a4c8643e",
        "ticker": "458730",
        "group_id": "3ae6cace-2a8c-44e6-a4d2-ee17c67b07c4",
        "exchange": None,
        "created_at": "2026-01-03T14:36:20.572862+00:00",
        "updated_at": "2026-01-03T14:36:20.572862+00:00",
    },
    {
        "id": "ca4a35e4-7bd8-4a4e-a22d-4fa7b587acc8",
        "ticker": "0052D0",
        "group_id": "2ca48716-5990-4612-8314-e7336c1a2d61",
        "exchange": None,
        "created_at": "2026-01-03T14:38:52.232698+00:00",
        "updated_at": "2026-01-03T14:38:52.232698+00:00",
    },
    {
        "id": "6c310b3c-04fc-4559-8b5a-a78328f5b30a",
        "ticker": "360750",
        "group_id": "e554b7d1-8cd1-44e1-8d8c-9bc8532c0612",
        "exchange": None,
        "created_at": "2026-01-03T14:37:20.363543+00:00",
        "updated_at": "2026-02-01T02:04:24.859015+00:00",
    },
    {
        "id": "ecc882f3-5259-47a5-a275-ca8b9ded625b",
        "ticker": "368590",
        "group_id": "a1ae778b-7705-405b-b0b2-31de492e0812",
        "exchange": None,
        "created_at": "2026-01-03T14:36:50.051342+00:00",
        "updated_at": "2026-02-01T02:04:47.582725+00:00",
    },
    {
        "id": "05cd6e63-0322-4d3f-9ab8-1af2f0451d60",
        "ticker": "VYM",
        "group_id": "3ae6cace-2a8c-44e6-a4d2-ee17c67b07c4",
        "exchange": "AMEX",
        "created_at": "2026-01-04T04:33:06.392514+00:00",
        "updated_at": "2026-02-19T05:32:30.468452+00:00",
    },
    {
        "id": "6d116b77-18a3-46ec-b4df-84bcd7646e23",
        "ticker": "SCHD",
        "group_id": "3ae6cace-2a8c-44e6-a4d2-ee17c67b07c4",
        "exchange": "AMEX",
        "created_at": "2026-01-04T04:33:17.834647+00:00",
        "updated_at": "2026-02-19T05:32:30.80956+00:00",
    },
    {
        "id": "b132ae60-bf85-44a6-85d7-d68978d8679f",
        "ticker": "QQQ",
        "group_id": "a1ae778b-7705-405b-b0b2-31de492e0812",
        "exchange": "NASD",
        "created_at": "2026-01-04T04:32:51.055318+00:00",
        "updated_at": "2026-02-19T05:32:31.041109+00:00",
    },
    {
        "id": "63c8ad5d-aaff-4e57-9933-a2a10383720d",
        "ticker": "VOO",
        "group_id": "e554b7d1-8cd1-44e1-8d8c-9bc8532c0612",
        "exchange": "AMEX",
        "created_at": "2026-01-04T04:33:35.112367+00:00",
        "updated_at": "2026-02-19T05:32:32.788447+00:00",
    },
    {
        "id": "12cde235-9a7d-402d-bb7d-e0872fb6e316",
        "ticker": "SPY",
        "group_id": "e554b7d1-8cd1-44e1-8d8c-9bc8532c0612",
        "exchange": "AMEX",
        "created_at": "2026-01-04T04:32:39.150998+00:00",
        "updated_at": "2026-03-05T02:30:48.589797+00:00",
    },
]

ACCOUNTS = [
    {
        "id": "5aa9c13b-1ac7-4c0d-abe2-a9ee715b0f84",
        "name": "TOSS",
        "cash_balance": "2849074",
        "created_at": "2026-01-03T13:21:44.873677+00:00",
        "updated_at": "2026-03-18T00:23:50.347457+00:00",
    },
    {
        "id": "89ce083b-6929-4db1-a385-d7af0225007a",
        "name": "IRP",
        "cash_balance": "11387",
        "created_at": "2026-01-03T13:23:15.586536+00:00",
        "updated_at": "2026-03-18T00:23:50.393545+00:00",
    },
    {
        "id": "c82018d6-5da2-4baf-bd48-05a45fba737d",
        "name": "ISA",
        "cash_balance": "17925",
        "created_at": "2026-01-03T13:21:40.015836+00:00",
        "updated_at": "2026-03-18T00:23:50.440446+00:00",
    },
]

HOLDINGS = [
    {
        "id": "388aff0c-0de4-4cc6-84fe-3272949e8ee5",
        "account_id": "c82018d6-5da2-4baf-bd48-05a45fba737d",
        "stock_id": "3907f51f-343c-4f01-8d91-36ba26498258",
        "quantity": "893",
        "created_at": "2026-01-03T14:35:34.960783+00:00",
        "updated_at": "2026-03-18T00:22:34.541154+00:00",
    },
    {
        "id": "ebac4c4d-f655-4388-b9c5-15aa473dc33f",
        "account_id": "c82018d6-5da2-4baf-bd48-05a45fba737d",
        "stock_id": "2b39913c-402d-4bac-97c3-2fb3a4c8643e",
        "quantity": "625",
        "created_at": "2026-01-03T14:36:28.911885+00:00",
        "updated_at": "2026-03-18T00:22:34.541154+00:00",
    },
    {
        "id": "cf50d9bb-fc0b-4ef1-b8f2-cccf832f2f3c",
        "account_id": "c82018d6-5da2-4baf-bd48-05a45fba737d",
        "stock_id": "6c310b3c-04fc-4559-8b5a-a78328f5b30a",
        "quantity": "27",
        "created_at": "2026-01-03T14:37:22.380415+00:00",
        "updated_at": "2026-03-18T00:22:34.541154+00:00",
    },
    {
        "id": "bece51ec-cfaf-4809-8303-f7f848c77bc0",
        "account_id": "c82018d6-5da2-4baf-bd48-05a45fba737d",
        "stock_id": "ecc882f3-5259-47a5-a275-ca8b9ded625b",
        "quantity": "484",
        "created_at": "2026-01-03T14:37:52.37008+00:00",
        "updated_at": "2026-03-18T00:22:34.541154+00:00",
    },
    {
        "id": "fb7a26f9-6ac4-4166-b825-89628023c258",
        "account_id": "c82018d6-5da2-4baf-bd48-05a45fba737d",
        "stock_id": "ca4a35e4-7bd8-4a4e-a22d-4fa7b587acc8",
        "quantity": "903",
        "created_at": "2026-01-03T14:38:55.815721+00:00",
        "updated_at": "2026-03-18T00:22:34.541154+00:00",
    },
    {
        "id": "bc6d7748-a29a-4eaf-b37a-c35c5ffdba38",
        "account_id": "89ce083b-6929-4db1-a385-d7af0225007a",
        "stock_id": "ca4a35e4-7bd8-4a4e-a22d-4fa7b587acc8",
        "quantity": "292",
        "created_at": "2026-01-04T05:35:08.934563+00:00",
        "updated_at": "2026-03-18T00:23:30.927451+00:00",
    },
    {
        "id": "2609789b-d013-4ffc-ba83-0b0c80866691",
        "account_id": "89ce083b-6929-4db1-a385-d7af0225007a",
        "stock_id": "3907f51f-343c-4f01-8d91-36ba26498258",
        "quantity": "360",
        "created_at": "2026-01-04T05:36:36.274438+00:00",
        "updated_at": "2026-03-18T00:23:30.927451+00:00",
    },
    {
        "id": "d60e118e-10eb-4a26-b20b-11c46e03731d",
        "account_id": "89ce083b-6929-4db1-a385-d7af0225007a",
        "stock_id": "6c310b3c-04fc-4559-8b5a-a78328f5b30a",
        "quantity": "648",
        "created_at": "2026-01-04T05:36:54.534377+00:00",
        "updated_at": "2026-03-18T00:23:30.927451+00:00",
    },
    {
        "id": "237553be-8c62-4800-8072-3c9496d7ab37",
        "account_id": "89ce083b-6929-4db1-a385-d7af0225007a",
        "stock_id": "2b39913c-402d-4bac-97c3-2fb3a4c8643e",
        "quantity": "118",
        "created_at": "2026-01-04T05:37:06.601842+00:00",
        "updated_at": "2026-03-18T00:23:30.927451+00:00",
    },
    {
        "id": "4bd3ac95-cb55-4b5a-b9b8-2beb903ff7a7",
        "account_id": "5aa9c13b-1ac7-4c0d-abe2-a9ee715b0f84",
        "stock_id": "3907f51f-343c-4f01-8d91-36ba26498258",
        "quantity": "978",
        "created_at": "2026-01-04T04:32:09.753838+00:00",
        "updated_at": "2026-03-18T00:24:08.522487+00:00",
    },
    {
        "id": "7ca9edcb-8d68-45ca-a578-b6d7a8d923d8",
        "account_id": "5aa9c13b-1ac7-4c0d-abe2-a9ee715b0f84",
        "stock_id": "ca4a35e4-7bd8-4a4e-a22d-4fa7b587acc8",
        "quantity": "507",
        "created_at": "2026-01-04T04:32:30.338812+00:00",
        "updated_at": "2026-03-18T00:24:08.522487+00:00",
    },
    {
        "id": "b2bddc4b-51e9-4105-bdbe-034dc0129e67",
        "account_id": "5aa9c13b-1ac7-4c0d-abe2-a9ee715b0f84",
        "stock_id": "12cde235-9a7d-402d-bb7d-e0872fb6e316",
        "quantity": "6.546644",
        "created_at": "2026-01-04T04:32:45.16918+00:00",
        "updated_at": "2026-03-18T00:24:08.522487+00:00",
    },
    {
        "id": "50f735cc-df67-485b-8b3c-512c88e95682",
        "account_id": "5aa9c13b-1ac7-4c0d-abe2-a9ee715b0f84",
        "stock_id": "b132ae60-bf85-44a6-85d7-d68978d8679f",
        "quantity": "36.693167",
        "created_at": "2026-01-04T04:32:58.112081+00:00",
        "updated_at": "2026-03-18T00:24:08.522487+00:00",
    },
    {
        "id": "63df1f63-705d-4701-9b6c-01ff03a4ee0b",
        "account_id": "5aa9c13b-1ac7-4c0d-abe2-a9ee715b0f84",
        "stock_id": "05cd6e63-0322-4d3f-9ab8-1af2f0451d60",
        "quantity": "33.180914",
        "created_at": "2026-01-04T04:33:10.660648+00:00",
        "updated_at": "2026-03-18T00:24:08.522487+00:00",
    },
    {
        "id": "bd265c64-5270-4b99-8c5a-4f6190edff8f",
        "account_id": "5aa9c13b-1ac7-4c0d-abe2-a9ee715b0f84",
        "stock_id": "6d116b77-18a3-46ec-b4df-84bcd7646e23",
        "quantity": "343.405634",
        "created_at": "2026-01-04T04:33:22.60127+00:00",
        "updated_at": "2026-03-18T00:24:08.522487+00:00",
    },
    {
        "id": "40235448-f3b2-48b3-8f4b-dcb9377fa173",
        "account_id": "5aa9c13b-1ac7-4c0d-abe2-a9ee715b0f84",
        "stock_id": "63c8ad5d-aaff-4e57-9933-a2a10383720d",
        "quantity": "4.575702",
        "created_at": "2026-01-04T04:33:41.800253+00:00",
        "updated_at": "2026-03-18T00:24:08.522487+00:00",
    },
    {
        "id": "97512897-91c7-4631-b39b-38e5bf7c36d8",
        "account_id": "5aa9c13b-1ac7-4c0d-abe2-a9ee715b0f84",
        "stock_id": "ecc882f3-5259-47a5-a275-ca8b9ded625b",
        "quantity": "219",
        "created_at": "2026-01-13T01:39:44.926883+00:00",
        "updated_at": "2026-03-18T00:24:08.522487+00:00",
    },
    {
        "id": "67792fd2-38c2-4709-9094-dd2c60f798a8",
        "account_id": "5aa9c13b-1ac7-4c0d-abe2-a9ee715b0f84",
        "stock_id": "6c310b3c-04fc-4559-8b5a-a78328f5b30a",
        "quantity": "120",
        "created_at": "2026-01-13T01:39:54.832277+00:00",
        "updated_at": "2026-03-18T00:24:08.522487+00:00",
    },
    {
        "id": "d3b55a3f-0e7e-4af6-ad8f-b54794ca9f21",
        "account_id": "5aa9c13b-1ac7-4c0d-abe2-a9ee715b0f84",
        "stock_id": "2b39913c-402d-4bac-97c3-2fb3a4c8643e",
        "quantity": "309",
        "created_at": "2026-01-13T01:40:04.455045+00:00",
        "updated_at": "2026-03-18T00:24:08.522487+00:00",
    },
]

DEPOSITS = [
    {
        "id": "8a867a59-5419-4ea9-bed4-007462a3a72e",
        "amount": "86060000",
        "deposit_date": "2021-01-06",
        "note": None,
        "created_at": "2026-01-04T07:10:01.371653+00:00",
        "updated_at": "2026-01-04T07:10:01.371653+00:00",
    },
    {
        "id": "bfac02de-bf66-44be-9673-7c58e84d8213",
        "amount": "50000000",
        "deposit_date": "2026-01-08",
        "note": "아내 투자",
        "created_at": "2026-01-13T01:35:33.080694+00:00",
        "updated_at": "2026-01-13T01:35:33.080694+00:00",
    },
    {
        "id": "0d3fb425-1c3d-4560-bbf9-fb55eb5e844a",
        "amount": "1670000",
        "deposit_date": "2026-01-04",
        "note": None,
        "created_at": "2026-01-04T07:14:08.463655+00:00",
        "updated_at": "2026-01-13T01:36:29.082011+00:00",
    },
    {
        "id": "4fec8d9c-16ac-43a6-89b9-f665abb5fe44",
        "amount": "500000",
        "deposit_date": "2026-01-20",
        "note": None,
        "created_at": "2026-01-20T04:05:00.138441+00:00",
        "updated_at": "2026-01-20T04:05:00.138441+00:00",
    },
    {
        "id": "dc9cb8b1-29db-4a9e-8c96-e6ddd4ceca81",
        "amount": "1037729",
        "deposit_date": "2026-02-03",
        "note": "채권 만기 상환",
        "created_at": "2026-02-04T12:32:34.989255+00:00",
        "updated_at": "2026-02-04T12:32:34.989255+00:00",
    },
    {
        "id": "e6008435-453e-46c8-bf4f-6f8d2b401c21",
        "amount": "999010",
        "deposit_date": "2026-02-10",
        "note": "채권 만기 상환",
        "created_at": "2026-02-12T02:26:23.638726+00:00",
        "updated_at": "2026-02-12T02:26:35.888942+00:00",
    },
    {
        "id": "1d9f6ecc-0a2b-4b03-bc48-f7c780bee6ae",
        "amount": "4000000",
        "deposit_date": "2026-02-12",
        "note": "학생지도비",
        "created_at": "2026-02-12T02:39:33.959695+00:00",
        "updated_at": "2026-02-12T02:39:33.959695+00:00",
    },
    {
        "id": "fc68aa8c-5b96-4078-bb9c-b68f54b2e664",
        "amount": "500000",
        "deposit_date": "2026-02-19",
        "note": None,
        "created_at": "2026-02-19T05:35:08.283841+00:00",
        "updated_at": "2026-02-19T05:35:08.283841+00:00",
    },
    {
        "id": "12623d71-d207-4db9-9f00-ac5ade801232",
        "amount": "998578",
        "deposit_date": "2026-02-27",
        "note": "채권 만기 상환",
        "created_at": "2026-02-27T13:28:52.348226+00:00",
        "updated_at": "2026-02-27T13:28:52.348226+00:00",
    },
    {
        "id": "fb15378e-bcf0-445e-811a-2fb89b638111",
        "amount": "500000",
        "deposit_date": "2026-03-17",
        "note": None,
        "created_at": "2026-03-20T12:11:27.207507+00:00",
        "updated_at": "2026-03-20T12:11:27.207507+00:00",
    },
]


def main():
    db_path = ".data/portfolio.db"
    if os.path.exists(db_path):
        print(f"WARNING: {db_path} already exists. Will be overwritten.")
        os.remove(db_path)

    print(f"Initializing SQLite at {db_path}...")
    init_db(db_path)

    with db.atomic():
        print(f"  groups: {len(GROUPS)} rows")
        for row in GROUPS:
            GroupModel.create(
                id=UUID(row["id"]),
                name=row["name"],
                target_percentage=row["target_percentage"],
                created_at=parse_dt(row["created_at"]),
                updated_at=parse_dt(row["updated_at"]),
            )

        print(f"  stocks: {len(STOCKS)} rows")
        for row in STOCKS:
            StockModel.create(
                id=UUID(row["id"]),
                ticker=row["ticker"],
                group=UUID(row["group_id"]),
                exchange=row["exchange"],
                created_at=parse_dt(row["created_at"]),
                updated_at=parse_dt(row["updated_at"]),
            )

        print(f"  accounts: {len(ACCOUNTS)} rows")
        for row in ACCOUNTS:
            AccountModel.create(
                id=UUID(row["id"]),
                name=row["name"],
                cash_balance=Decimal(row["cash_balance"]),
                created_at=parse_dt(row["created_at"]),
                updated_at=parse_dt(row["updated_at"]),
            )

        print(f"  holdings: {len(HOLDINGS)} rows")
        for row in HOLDINGS:
            HoldingModel.create(
                id=UUID(row["id"]),
                account=UUID(row["account_id"]),
                stock=UUID(row["stock_id"]),
                quantity=Decimal(row["quantity"]),
                created_at=parse_dt(row["created_at"]),
                updated_at=parse_dt(row["updated_at"]),
            )

        print(f"  deposits: {len(DEPOSITS)} rows")
        for row in DEPOSITS:
            DepositModel.create(
                id=UUID(row["id"]),
                amount=Decimal(row["amount"]),
                deposit_date=row["deposit_date"],
                note=row.get("note"),
                created_at=parse_dt(row["created_at"]),
                updated_at=parse_dt(row["updated_at"]),
            )

    print("\nVerification:")
    print(f"  groups: {GroupModel.select().count()}")
    print(f"  stocks: {StockModel.select().count()}")
    print(f"  accounts: {AccountModel.select().count()}")
    print(f"  holdings: {HoldingModel.select().count()}")
    print(f"  deposits: {DepositModel.select().count()}")
    print(f"\nDone! SQLite database saved to: {db_path}")


if __name__ == "__main__":
    main()
