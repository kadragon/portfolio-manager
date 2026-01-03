"""Tests for Rich-based holding add flow."""

from datetime import datetime
from decimal import Decimal
from uuid import uuid4
from unittest.mock import MagicMock

from rich.console import Console

from portfolio_manager.cli.rich_holdings import add_holding_flow
from portfolio_manager.models import Account, Group, Holding, Stock


def test_add_holding_flow_creates_holding_and_reports_quantity():
    """Should create holding and render confirmation."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    holding = Holding(
        id=uuid4(),
        account_id=account.id,
        stock_id=uuid4(),
        quantity=Decimal("5.5"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.create.return_value = holding

    add_holding_flow(
        console,
        repo,
        account,
        prompt_stock=lambda: holding.stock_id,
        prompt_quantity=lambda: Decimal("5.5"),
    )

    repo.create.assert_called_once_with(
        account_id=account.id,
        stock_id=holding.stock_id,
        quantity=Decimal("5.5"),
    )
    output = console.export_text()
    assert "5.5" in output


def test_add_holding_flow_resolves_ticker_to_stock_id():
    """Should resolve ticker to stock id when adding holding."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    stock_repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    stock_id = uuid4()
    stock_repo.get_by_ticker.return_value = Stock(
        id=stock_id,
        ticker="310970",
        group_id=uuid4(),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    holding = Holding(
        id=uuid4(),
        account_id=account.id,
        stock_id=stock_id,
        quantity=Decimal("3"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.create.return_value = holding

    add_holding_flow(
        console,
        repo,
        account,
        prompt_stock=lambda: "310970",
        prompt_quantity=lambda: Decimal("3"),
        stock_repository=stock_repo,
    )

    stock_repo.get_by_ticker.assert_called_once_with("310970")
    repo.create.assert_called_once_with(
        account_id=account.id,
        stock_id=stock_id,
        quantity=Decimal("3"),
    )


def test_add_holding_flow_creates_stock_when_missing():
    """Should create a stock when ticker is missing."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    stock_repo = MagicMock()
    group_repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    group_id = uuid4()
    group_repo.list_all.return_value = [
        Group(
            id=group_id,
            name="Tech",
            created_at=datetime.now(),
            updated_at=datetime.now(),
        )
    ]
    stock_repo.get_by_ticker.return_value = None
    stock_repo.create.return_value = Stock(
        id=uuid4(),
        ticker="310970",
        group_id=group_id,
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    holding = Holding(
        id=uuid4(),
        account_id=account.id,
        stock_id=stock_repo.create.return_value.id,
        quantity=Decimal("2"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.create.return_value = holding

    add_holding_flow(
        console,
        repo,
        account,
        prompt_stock=lambda: "310970",
        prompt_quantity=lambda: Decimal("2"),
        stock_repository=stock_repo,
        group_repository=group_repo,
        group_chooser=lambda **_: group_id,
    )

    stock_repo.create.assert_called_once_with("310970", group_id)
    repo.create.assert_called_once_with(
        account_id=account.id,
        stock_id=stock_repo.create.return_value.id,
        quantity=Decimal("2"),
    )


def test_add_holding_flow_creates_group_when_missing():
    """Should create a group when none exist."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    stock_repo = MagicMock()
    group_repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    group_repo.list_all.return_value = []
    group_repo.create.return_value = Group(
        id=uuid4(),
        name="Auto",
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    stock_repo.get_by_ticker.return_value = None
    stock_repo.create.return_value = Stock(
        id=uuid4(),
        ticker="310970",
        group_id=group_repo.create.return_value.id,
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.create.return_value = Holding(
        id=uuid4(),
        account_id=account.id,
        stock_id=stock_repo.create.return_value.id,
        quantity=Decimal("2"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )

    add_holding_flow(
        console,
        repo,
        account,
        prompt_stock=lambda: "310970",
        prompt_quantity=lambda: Decimal("2"),
        stock_repository=stock_repo,
        group_repository=group_repo,
        group_chooser=lambda **_: None,
        prompt_group_name=lambda: "Auto",
    )

    group_repo.create.assert_called_once_with("Auto")
    stock_repo.create.assert_called_once_with(
        "310970", group_repo.create.return_value.id
    )
