from __future__ import annotations

from dataclasses import dataclass
from datetime import date, datetime, timezone
from decimal import Decimal
from pathlib import Path
from uuid import UUID, uuid4

import pytest
from fastapi import FastAPI
from fastapi.templating import Jinja2Templates
from fastapi.testclient import TestClient

from portfolio_manager.models import Account, Deposit, Group, Holding, Stock
from portfolio_manager.services.portfolio_service import (
    GroupHoldings,
    PortfolioSummary,
    StockHolding,
    StockHoldingWithPrice,
)
from portfolio_manager.web.app import _add_filters
from portfolio_manager.web.routes import (
    accounts,
    dashboard,
    deposits,
    groups,
    rebalance,
)


class FakeGroupRepository:
    def __init__(self, groups_data: list[Group]):
        self._groups = groups_data

    def list_all(self) -> list[Group]:
        return self._groups


class FakeStockRepository:
    def __init__(self, stocks_data: list[Stock]):
        self._stocks = stocks_data

    def list_all(self) -> list[Stock]:
        return self._stocks

    def list_by_group(self, group_id: UUID) -> list[Stock]:
        return [stock for stock in self._stocks if stock.group_id == group_id]


class FakeAccountRepository:
    def __init__(self, accounts_data: list[Account]):
        self._accounts = accounts_data

    def list_all(self) -> list[Account]:
        return self._accounts


class FakeHoldingRepository:
    def __init__(self, holdings_data: list[Holding]):
        self._holdings = holdings_data

    def list_by_account(self, account_id: UUID) -> list[Holding]:
        return [
            holding for holding in self._holdings if holding.account_id == account_id
        ]


class FakeDepositRepository:
    def __init__(self, deposits_data: list[Deposit]):
        self._deposits = deposits_data

    def list_all(self) -> list[Deposit]:
        return self._deposits

    def get_total(self) -> Decimal:
        return sum((deposit.amount for deposit in self._deposits), Decimal("0"))


@dataclass
class FakePortfolioService:
    summary: PortfolioSummary
    group_holdings: list[GroupHoldings]

    def get_portfolio_summary(
        self, *, include_change_rates: bool = True
    ) -> PortfolioSummary:
        return self.summary

    def get_holdings_by_group(self) -> list[GroupHoldings]:
        return self.group_holdings


class FakeKisAccountSyncService:
    def sync_account(self, **_: object) -> None:
        return None


class FakeContainer:
    def __init__(self) -> None:
        now = datetime.now(timezone.utc)

        self.group = Group(
            id=uuid4(),
            name="국내 주식",
            created_at=now,
            updated_at=now,
            target_percentage=0.0,
        )
        self.stock = Stock(
            id=uuid4(),
            ticker="005930",
            group_id=self.group.id,
            created_at=now,
            updated_at=now,
            exchange=None,
        )
        self.account = Account(
            id=uuid4(),
            name="메인 계좌",
            cash_balance=Decimal("300000"),
            created_at=now,
            updated_at=now,
        )
        self.holding = Holding(
            id=uuid4(),
            account_id=self.account.id,
            stock_id=self.stock.id,
            quantity=Decimal("10"),
            created_at=now,
            updated_at=now,
        )
        self.deposit = Deposit(
            id=uuid4(),
            amount=Decimal("900000"),
            deposit_date=date(2026, 1, 5),
            created_at=now,
            updated_at=now,
            note="초기 입금",
        )

        stock_holding_with_price = StockHoldingWithPrice(
            stock=self.stock,
            quantity=Decimal("10"),
            price=Decimal("70000"),
            currency="KRW",
            name="삼성전자",
            value_krw=Decimal("700000"),
        )

        self.portfolio_service = FakePortfolioService(
            summary=PortfolioSummary(
                holdings=[(self.group, stock_holding_with_price)],
                total_value=Decimal("700000"),
                total_stock_value=Decimal("700000"),
                total_cash_balance=Decimal("300000"),
                total_assets=Decimal("1000000"),
                total_invested=Decimal("900000"),
                return_rate=Decimal("11.1"),
                first_deposit_date=self.deposit.deposit_date,
                annualized_return_rate=Decimal("18.4"),
            ),
            group_holdings=[
                GroupHoldings(
                    group=self.group,
                    stock_holdings=[
                        StockHolding(stock=self.stock, quantity=Decimal("10"))
                    ],
                )
            ],
        )

        self.group_repository = FakeGroupRepository([self.group])
        self.stock_repository = FakeStockRepository([self.stock])
        self.account_repository = FakeAccountRepository([self.account])
        self.holding_repository = FakeHoldingRepository([self.holding])
        self.deposit_repository = FakeDepositRepository([self.deposit])

        self.price_service = object()
        self.order_client = object()
        self.execution_repository = None
        self.kis_account_sync_service = FakeKisAccountSyncService()
        self.kis_cano = "12345678"
        self.kis_acnt_prdt_cd = "01"

    def get_portfolio_service(self) -> FakePortfolioService:
        return self.portfolio_service


@pytest.fixture
def fake_container() -> FakeContainer:
    return FakeContainer()


@pytest.fixture
def app(fake_container: FakeContainer) -> FastAPI:
    app_instance = FastAPI()

    templates_dir = (
        Path(__file__).resolve().parents[2]
        / "src"
        / "portfolio_manager"
        / "web"
        / "templates"
    )
    templates = Jinja2Templates(directory=str(templates_dir))
    _add_filters(templates)

    app_instance.state.container = fake_container
    app_instance.state.templates = templates

    app_instance.include_router(dashboard.router)
    app_instance.include_router(groups.router)
    app_instance.include_router(accounts.router)
    app_instance.include_router(deposits.router)
    app_instance.include_router(rebalance.router)

    return app_instance


@pytest.fixture
def client(app: FastAPI) -> TestClient:
    return TestClient(app)
