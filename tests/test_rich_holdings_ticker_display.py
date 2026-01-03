"""Tests for Rich holdings ticker display."""

from decimal import Decimal
from datetime import datetime
from uuid import uuid4
from unittest.mock import MagicMock

from rich.console import Console

from portfolio_manager.cli.rich_holdings import render_holdings_for_account
from portfolio_manager.models import Account, Holding


def test_render_holdings_for_account_displays_ticker():
    """Should render ticker instead of raw stock ID when mapping provided."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    stock_id = uuid4()
    holdings = [
        Holding(
            id=uuid4(),
            account_id=account.id,
            stock_id=stock_id,
            quantity=Decimal("5.5"),
            created_at=datetime.now(),
            updated_at=datetime.now(),
        )
    ]
    repo.list_by_account.return_value = holdings

    render_holdings_for_account(
        console, repo, account, stock_lookup=lambda _stock_id: "AAPL"
    )

    output = console.export_text()
    assert "AAPL" in output
