"""Tests for Rich-based holdings list rendering."""

from decimal import Decimal
from datetime import datetime
from uuid import uuid4
from unittest.mock import MagicMock

from rich.console import Console

from portfolio_manager.cli.rich_holdings import render_holdings_for_account
from portfolio_manager.models import Account, Holding


def test_render_holdings_for_account_lists_holdings():
    """Should render holdings for the selected account."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    holdings = [
        Holding(
            id=uuid4(),
            account_id=account.id,
            stock_id=uuid4(),
            quantity=Decimal("5.5"),
            created_at=datetime.now(),
            updated_at=datetime.now(),
        ),
        Holding(
            id=uuid4(),
            account_id=account.id,
            stock_id=uuid4(),
            quantity=Decimal("10"),
            created_at=datetime.now(),
            updated_at=datetime.now(),
        ),
    ]
    repo.list_by_account.return_value = holdings

    render_holdings_for_account(console, repo, account)

    repo.list_by_account.assert_called_once_with(account.id)
    output = console.export_text()
    assert "5.5" in output
    assert "10" in output
