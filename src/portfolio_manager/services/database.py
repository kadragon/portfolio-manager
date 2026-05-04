"""SQLite database module using Peewee ORM."""

import os
from pathlib import Path
from decimal import Decimal
from uuid import uuid4

from portfolio_manager.core.time import now_kst

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

    def save(self, *args, **kwargs):
        if hasattr(self, "updated_at") and not kwargs.get("force_insert", False):
            self.updated_at = now_kst()
        return super().save(*args, **kwargs)

    class Meta:
        database = db


class GroupModel(BaseModel):
    id = UUIDField(primary_key=True, default=uuid4)
    name = TextField()
    target_percentage = FloatField(default=0.0)
    created_at = DateTimeField(default=now_kst)
    updated_at = DateTimeField(default=now_kst)

    class Meta:
        table_name = "groups"


class StockModel(BaseModel):
    id = UUIDField(primary_key=True, default=uuid4)
    ticker = TextField()
    name = TextField(default="")
    group = ForeignKeyField(
        GroupModel, column_name="group_id", backref="stocks", on_delete="CASCADE"
    )
    exchange = TextField(null=True)
    created_at = DateTimeField(default=now_kst)
    updated_at = DateTimeField(default=now_kst)

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
    kis_account_no = TextField(null=True)
    kis_api_key_id = IntegerField(null=True)
    created_at = DateTimeField(default=now_kst)
    updated_at = DateTimeField(default=now_kst)

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
    created_at = DateTimeField(default=now_kst)
    updated_at = DateTimeField(default=now_kst)

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
    created_at = DateTimeField(default=now_kst)
    updated_at = DateTimeField(default=now_kst)

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
    created_at = DateTimeField(default=now_kst)
    updated_at = DateTimeField(default=now_kst)

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
    created_at = DateTimeField(default=now_kst)

    class Meta:
        table_name = "order_executions"
        indexes = ((("created_at",), False),)


class InvestorFlowModel(BaseModel):
    id = UUIDField(primary_key=True, default=uuid4)
    ticker = TextField()
    flow_date = DateField()
    foreign_net_qty = IntegerField(default=0)
    institution_net_qty = IntegerField(default=0)
    individual_net_qty = IntegerField(default=0)
    foreign_net_krw = IntegerField(default=0)
    institution_net_krw = IntegerField(default=0)
    individual_net_krw = IntegerField(default=0)
    created_at = DateTimeField(default=now_kst)
    updated_at = DateTimeField(default=now_kst)

    class Meta:
        table_name = "investor_flows"
        indexes = (
            (("ticker", "flow_date"), True),
            (("ticker",), False),
        )


ALL_MODELS = [
    GroupModel,
    StockModel,
    AccountModel,
    HoldingModel,
    DepositModel,
    StockPriceModel,
    OrderExecutionModel,
    InvestorFlowModel,
]


def _default_db_path() -> str:
    """Return the default database path as an absolute path."""
    env_path = os.environ.get("PORTFOLIO_DB_PATH")
    if env_path:
        return str(Path(env_path).resolve())
    # database.py → services → portfolio_manager → src → project root
    project_root = Path(__file__).resolve().parents[3]
    db_path = project_root / ".data" / "portfolio.db"
    if not project_root.joinpath("pyproject.toml").exists():
        raise RuntimeError(
            f"Cannot locate project root (resolved to {project_root}). "
            "Set the PORTFOLIO_DB_PATH environment variable."
        )
    return str(db_path)


def init_db(db_path: str | None = None) -> SqliteDatabase:
    """Initialize the SQLite database and create tables if needed."""
    if db_path is None:
        db_path = _default_db_path()
    os.makedirs(os.path.dirname(db_path), exist_ok=True)
    db.init(
        db_path,
        pragmas={
            "journal_mode": "wal",
            "foreign_keys": 1,
        },
    )
    db.create_tables(ALL_MODELS)
    _run_migrations(db)
    return db


def _run_migrations(database: SqliteDatabase) -> None:
    """Apply schema migrations for columns added after initial release."""
    from playhouse.migrate import SqliteMigrator, migrate

    migrator = SqliteMigrator(database)
    account_columns = {col.name for col in database.get_columns("accounts")}
    if "kis_account_no" not in account_columns:
        migrate(migrator.add_column("accounts", "kis_account_no", TextField(null=True)))
    if "kis_api_key_id" not in account_columns:
        migrate(
            migrator.add_column("accounts", "kis_api_key_id", IntegerField(null=True))
        )

    stock_columns = {col.name for col in database.get_columns("stocks")}
    if "name" not in stock_columns:
        migrate(migrator.add_column("stocks", "name", TextField(default="")))
        database.execute_sql(
            """
            UPDATE stocks SET name = (
                SELECT sp.name FROM stock_prices sp
                WHERE sp.ticker = stocks.ticker AND sp.name <> ''
                ORDER BY sp.price_date DESC LIMIT 1
            )
            WHERE (name IS NULL OR name = '')
              AND EXISTS (
                  SELECT 1 FROM stock_prices sp
                  WHERE sp.ticker = stocks.ticker AND sp.name <> ''
              )
            """
        )


def close_db() -> None:
    """Close the database connection if open."""
    if not db.is_closed():
        db.close()
