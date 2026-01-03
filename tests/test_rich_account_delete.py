"""Tests for Rich-based account delete flow."""

from datetime import datetime
from decimal import Decimal
from uuid import uuid4
from unittest.mock import MagicMock

from rich.console import Console

from portfolio_manager.cli.rich_accounts import delete_account_flow
from portfolio_manager.models import Account


def test_delete_account_flow_removes_account_and_reports_name():
    """Should delete account and render confirmation."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    holding_repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )

    delete_account_flow(console, repo, holding_repo, account, confirm=lambda: True)

    repo.delete_with_holdings.assert_called_once_with(account.id, holding_repo)
    output = console.export_text()
    assert "Brokerage" in output
