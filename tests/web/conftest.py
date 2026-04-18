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
    insights,
    rebalance,
)


class _UnsetNote:
    pass


_NOTE_UNSET = _UnsetNote()


class FakeGroupRepository:
    def __init__(self, groups_data: list[Group]):
        self._groups = groups_data

    def list_all(self) -> list[Group]:
        return self._groups

    def create(self, name: str, target_percentage: float = 0.0) -> Group:
        now = datetime.now(timezone.utc)
        group = Group(
            id=uuid4(),
            name=name,
            created_at=now,
            updated_at=now,
            target_percentage=target_percentage,
        )
        self._groups.insert(0, group)
        return group

    def update(
        self,
        group_id: UUID,
        name: str | None = None,
        target_percentage: float | None = None,
    ) -> Group:
        for idx, group in enumerate(self._groups):
            if group.id == group_id:
                updated = Group(
                    id=group.id,
                    name=group.name if name is None else name,
                    created_at=group.created_at,
                    updated_at=datetime.now(timezone.utc),
                    target_percentage=(
                        group.target_percentage
                        if target_percentage is None
                        else target_percentage
                    ),
                )
                self._groups[idx] = updated
                return updated
        raise ValueError("group not found")

    def delete(self, group_id: UUID) -> None:
        self._groups = [group for group in self._groups if group.id != group_id]


class FakeStockRepository:
    def __init__(self, stocks_data: list[Stock]):
        self._stocks = stocks_data

    def list_all(self) -> list[Stock]:
        return self._stocks

    def list_by_group(self, group_id: UUID) -> list[Stock]:
        return [stock for stock in self._stocks if stock.group_id == group_id]

    def get_by_id(self, stock_id: UUID) -> Stock | None:
        for stock in self._stocks:
            if stock.id == stock_id:
                return stock
        return None

    def get_by_ticker(self, ticker: str) -> Stock | None:
        for stock in self._stocks:
            if stock.ticker == ticker:
                return stock
        return None

    def create(self, ticker: str, group_id: UUID, *, name: str = "") -> Stock:
        now = datetime.now(timezone.utc)
        stock = Stock(
            id=uuid4(),
            ticker=ticker,
            group_id=group_id,
            created_at=now,
            updated_at=now,
            exchange=None,
            name=name,
        )
        self._stocks.insert(0, stock)
        return stock

    def update(self, stock_id: UUID, ticker: str) -> Stock:
        for idx, stock in enumerate(self._stocks):
            if stock.id == stock_id:
                updated = Stock(
                    id=stock.id,
                    ticker=ticker,
                    group_id=stock.group_id,
                    created_at=stock.created_at,
                    updated_at=datetime.now(timezone.utc),
                    exchange=stock.exchange,
                )
                self._stocks[idx] = updated
                return updated
        raise ValueError("stock not found")

    def update_name(self, stock_id: UUID, name: str) -> Stock:
        for idx, stock in enumerate(self._stocks):
            if stock.id == stock_id:
                updated = Stock(
                    id=stock.id,
                    ticker=stock.ticker,
                    group_id=stock.group_id,
                    created_at=stock.created_at,
                    updated_at=datetime.now(timezone.utc),
                    exchange=stock.exchange,
                    name=name,
                )
                self._stocks[idx] = updated
                return updated
        raise ValueError("stock not found")

    def update_group(self, stock_id: UUID, group_id: UUID) -> Stock:
        for idx, stock in enumerate(self._stocks):
            if stock.id == stock_id:
                updated = Stock(
                    id=stock.id,
                    ticker=stock.ticker,
                    group_id=group_id,
                    created_at=stock.created_at,
                    updated_at=datetime.now(timezone.utc),
                    exchange=stock.exchange,
                )
                self._stocks[idx] = updated
                return updated
        raise ValueError("stock not found")

    def delete(self, stock_id: UUID) -> None:
        self._stocks = [stock for stock in self._stocks if stock.id != stock_id]


_FAKE_UNSET = object()


class FakeAccountRepository:
    def __init__(self, accounts_data: list[Account]):
        self._accounts = accounts_data

    def list_all(self) -> list[Account]:
        return self._accounts

    def create(self, *, name: str, cash_balance: Decimal) -> Account:
        now = datetime.now(timezone.utc)
        account = Account(
            id=uuid4(),
            name=name,
            cash_balance=cash_balance,
            created_at=now,
            updated_at=now,
        )
        self._accounts.insert(0, account)
        return account

    def update(
        self,
        *,
        account_id: UUID,
        name: str,
        cash_balance: Decimal,
        kis_account_no=_FAKE_UNSET,
        kis_api_key_id=_FAKE_UNSET,
    ) -> Account:
        for idx, account in enumerate(self._accounts):
            if account.id == account_id:
                updated = Account(
                    id=account.id,
                    name=name,
                    cash_balance=cash_balance,
                    created_at=account.created_at,
                    updated_at=datetime.now(timezone.utc),
                    kis_account_no=kis_account_no
                    if kis_account_no is not _FAKE_UNSET
                    else account.kis_account_no,
                    kis_api_key_id=kis_api_key_id
                    if kis_api_key_id is not _FAKE_UNSET
                    else account.kis_api_key_id,
                )
                self._accounts[idx] = updated
                return updated
        raise ValueError("account not found")

    def delete_with_holdings(self, account_id: UUID, holding_repository) -> None:
        holding_repository.delete_by_account(account_id)
        self._accounts = [a for a in self._accounts if a.id != account_id]


class FakeHoldingRepository:
    def __init__(self, holdings_data: list[Holding]):
        self._holdings = holdings_data

    def list_by_account(self, account_id: UUID) -> list[Holding]:
        return [
            holding for holding in self._holdings if holding.account_id == account_id
        ]

    def create(self, *, account_id: UUID, stock_id: UUID, quantity: Decimal) -> Holding:
        now = datetime.now(timezone.utc)
        holding = Holding(
            id=uuid4(),
            account_id=account_id,
            stock_id=stock_id,
            quantity=quantity,
            created_at=now,
            updated_at=now,
        )
        self._holdings.insert(0, holding)
        return holding

    def update(self, *, holding_id: UUID, quantity: Decimal) -> Holding:
        for idx, holding in enumerate(self._holdings):
            if holding.id == holding_id:
                updated = Holding(
                    id=holding.id,
                    account_id=holding.account_id,
                    stock_id=holding.stock_id,
                    quantity=quantity,
                    created_at=holding.created_at,
                    updated_at=datetime.now(timezone.utc),
                )
                self._holdings[idx] = updated
                return updated
        raise ValueError("holding not found")

    def bulk_update_by_account(
        self, account_id: UUID, updates: list[tuple[UUID, Decimal]]
    ) -> list[Holding]:
        if not updates:
            return []

        holding_ids = [holding_id for holding_id, _ in updates]
        if len(set(holding_ids)) != len(holding_ids):
            raise ValueError("duplicate holding_ids are not allowed")
        if any(quantity <= 0 for _, quantity in updates):
            raise ValueError("quantity must be greater than zero")

        index_by_id = {holding.id: idx for idx, holding in enumerate(self._holdings)}
        for holding_id, _ in updates:
            if holding_id not in index_by_id:
                raise ValueError("선택한 보유 내역이 해당 계좌에 속하지 않습니다.")
            if self._holdings[index_by_id[holding_id]].account_id != account_id:
                raise ValueError("선택한 보유 내역이 해당 계좌에 속하지 않습니다.")

        now = datetime.now(timezone.utc)
        updated_holdings: list[Holding] = []
        for holding_id, quantity in updates:
            idx = index_by_id[holding_id]
            current = self._holdings[idx]
            updated = Holding(
                id=current.id,
                account_id=current.account_id,
                stock_id=current.stock_id,
                quantity=quantity,
                created_at=current.created_at,
                updated_at=now,
            )
            self._holdings[idx] = updated
            updated_holdings.append(updated)
        return updated_holdings

    def delete(self, holding_id: UUID) -> None:
        self._holdings = [
            holding for holding in self._holdings if holding.id != holding_id
        ]

    def delete_by_account(self, account_id: UUID) -> None:
        self._holdings = [
            holding for holding in self._holdings if holding.account_id != account_id
        ]


class FakeDepositRepository:
    def __init__(self, deposits_data: list[Deposit]):
        self._deposits = deposits_data

    def list_all(self) -> list[Deposit]:
        return self._deposits

    def get_total(self) -> Decimal:
        return sum((deposit.amount for deposit in self._deposits), Decimal("0"))

    def get_by_date(self, deposit_date: date) -> Deposit | None:
        return next(
            (
                deposit
                for deposit in self._deposits
                if deposit.deposit_date == deposit_date
            ),
            None,
        )

    def create(
        self, *, amount: Decimal, deposit_date: date, note: str | None
    ) -> Deposit:
        now = datetime.now(timezone.utc)
        deposit = Deposit(
            id=uuid4(),
            amount=amount,
            deposit_date=deposit_date,
            created_at=now,
            updated_at=now,
            note=note,
        )
        self._deposits.insert(0, deposit)
        return deposit

    def update(
        self,
        *,
        deposit_id: UUID,
        amount: Decimal,
        deposit_date: date | None = None,
        note: str | None | _UnsetNote = _NOTE_UNSET,
    ) -> Deposit:
        for idx, deposit in enumerate(self._deposits):
            if deposit.id == deposit_id:
                next_note = deposit.note if isinstance(note, _UnsetNote) else note
                updated = Deposit(
                    id=deposit.id,
                    amount=amount,
                    deposit_date=deposit.deposit_date
                    if deposit_date is None
                    else deposit_date,
                    created_at=deposit.created_at,
                    updated_at=datetime.now(timezone.utc),
                    note=next_note,
                )
                self._deposits[idx] = updated
                return updated
        raise ValueError("deposit not found")

    def delete(self, deposit_id: UUID) -> None:
        self._deposits = [
            deposit for deposit in self._deposits if deposit.id != deposit_id
        ]


@dataclass
class FakePortfolioService:
    summary: PortfolioSummary
    group_holdings: list[GroupHoldings]

    def get_portfolio_summary(
        self,
        *,
        include_change_rates: bool = True,
        change_rate_periods: tuple[str, ...] | None = None,
    ) -> PortfolioSummary:
        return self.summary

    def get_holdings_by_group(self) -> list[GroupHoldings]:
        return self.group_holdings


class FakeKisAccountSyncService:
    def __init__(self) -> None:
        self.sync_exception: Exception | None = None
        self.sync_exception_unless_confirm: Exception | None = None
        self.validate_exception: Exception | None = None
        self.validated_accounts: list[tuple[str, str]] = []
        self.sync_calls: list[dict] = []

    def sync_account(self, **kwargs: object) -> None:
        self.sync_calls.append(kwargs)
        if self.sync_exception is not None:
            raise self.sync_exception
        if self.sync_exception_unless_confirm is not None and not kwargs.get(
            "allow_empty_snapshot"
        ):
            raise self.sync_exception_unless_confirm
        return None

    def validate_account(
        self, *, cano: str, acnt_prdt_cd: str, kis_balance_client=None
    ) -> None:
        self.validated_accounts.append((cano, acnt_prdt_cd))
        if self.validate_exception is not None:
            raise self.validate_exception


class FakeContainer:
    def __init__(self) -> None:
        now = datetime.now(timezone.utc)

        self.group = Group(
            id=uuid4(),
            name="국내성장",
            created_at=now,
            updated_at=now,
            target_percentage=35.0,
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
            kis_account_no="12345678-01",
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
        self.kis_client_sets: dict[int, object] = {}
        self.portfolio_insight_service: object | None = None

    def get_kis_client_set(self, key_id: int | None) -> object | None:
        effective_id = key_id or 1
        return self.kis_client_sets.get(effective_id)

    def get_portfolio_service(self) -> FakePortfolioService:
        return self.portfolio_service

    def get_portfolio_insight_service(self) -> object | None:
        return self.portfolio_insight_service


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
    app_instance.include_router(insights.router)

    return app_instance


@pytest.fixture
def client(app: FastAPI) -> TestClient:
    return TestClient(app)
