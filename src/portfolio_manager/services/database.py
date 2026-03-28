"""SQLite database module using Peewee ORM."""

import os
from datetime import datetime, timezone
from decimal import Decimal
from uuid import uuid4

from peewee import (
    DateField,
    DateTimeField,
    DecimalField,
    FloatField,
    ForeignKeyField,
    IntegerField,
    Model,
    SqliteDatabase,
    TextField,
    UUIDField,
)

db = SqliteDatabase(None)


class BaseModel(Model):
    """Base model with shared database reference."""

    class Meta:
        database = db


class GroupModel(BaseModel):
    id = UUIDField(primary_key=True, default=uuid4)
    name = TextField()
    target_percentage = FloatField(default=0.0)
    created_at = DateTimeField(default=lambda: datetime.now(timezone.utc))
    updated_at = DateTimeField(default=lambda: datetime.now(timezone.utc))

    class Meta:
        table_name = "groups"


class StockModel(BaseModel):
    id = UUIDField(primary_key=True, default=uuid4)
    ticker = TextField()
    group = ForeignKeyField(
        GroupModel, column_name="group_id", backref="stocks", on_delete="CASCADE"
    )
    exchange = TextField(null=True)
    created_at = DateTimeField(default=lambda: datetime.now(timezone.utc))
    updated_at = DateTimeField(default=lambda: datetime.now(timezone.utc))

    class Meta:
        table_name = "stocks"
        indexes = (
            (("group",), False),
            (("ticker",), False),
        )


class AccountModel(BaseModel):
    id = UUIDField(primary_key=True, default=uuid4)
    name = TextField()
    cash_balance = DecimalField(decimal_places=10, auto_round=False, default=Decimal(0))
    created_at = DateTimeField(default=lambda: datetime.now(timezone.utc))
    updated_at = DateTimeField(default=lambda: datetime.now(timezone.utc))

    class Meta:
        table_name = "accounts"


class HoldingModel(BaseModel):
    id = UUIDField(primary_key=True, default=uuid4)
    account = ForeignKeyField(
        AccountModel,
        column_name="account_id",
        backref="holdings",
        on_delete="CASCADE",
    )
    stock = ForeignKeyField(
        StockModel, column_name="stock_id", backref="holdings", on_delete="CASCADE"
    )
    quantity = DecimalField(decimal_places=10, auto_round=False, default=Decimal(0))
    created_at = DateTimeField(default=lambda: datetime.now(timezone.utc))
    updated_at = DateTimeField(default=lambda: datetime.now(timezone.utc))

    class Meta:
        table_name = "holdings"
        indexes = (
            (("account",), False),
            (("stock",), False),
        )


class DepositModel(BaseModel):
    id = UUIDField(primary_key=True, default=uuid4)
    amount = DecimalField(decimal_places=10, auto_round=False)
    deposit_date = DateField(unique=True)
    note = TextField(null=True)
    created_at = DateTimeField(default=lambda: datetime.now(timezone.utc))
    updated_at = DateTimeField(default=lambda: datetime.now(timezone.utc))

    class Meta:
        table_name = "deposits"


class StockPriceModel(BaseModel):
    id = UUIDField(primary_key=True, default=uuid4)
    ticker = TextField()
    price = DecimalField(decimal_places=10, auto_round=False)
    currency = TextField()
    name = TextField(default="")
    exchange = TextField(null=True)
    price_date = DateField()
    created_at = DateTimeField(default=lambda: datetime.now(timezone.utc))
    updated_at = DateTimeField(default=lambda: datetime.now(timezone.utc))

    class Meta:
        table_name = "stock_prices"
        indexes = (
            (("ticker", "price_date"), True),
            (("ticker",), False),
        )


class OrderExecutionModel(BaseModel):
    id = UUIDField(primary_key=True, default=uuid4)
    ticker = TextField()
    side = TextField()
    quantity = IntegerField()
    currency = TextField()
    exchange = TextField(null=True)
    status = TextField()
    message = TextField(default="")
    raw_response = TextField(null=True)
    created_at = DateTimeField(default=lambda: datetime.now(timezone.utc))

    class Meta:
        table_name = "order_executions"
        indexes = ((("created_at",), False),)


ALL_MODELS = [
    GroupModel,
    StockModel,
    AccountModel,
    HoldingModel,
    DepositModel,
    StockPriceModel,
    OrderExecutionModel,
]


def init_db(db_path: str = ".data/portfolio.db") -> SqliteDatabase:
    """Initialize the SQLite database and create tables if needed."""
    os.makedirs(os.path.dirname(db_path), exist_ok=True)
    db.init(
        db_path,
        pragmas={
            "journal_mode": "wal",
            "foreign_keys": 1,
        },
    )
    db.create_tables(ALL_MODELS)
    return db


def close_db() -> None:
    """Close the database connection if open."""
    if not db.is_closed():
        db.close()
