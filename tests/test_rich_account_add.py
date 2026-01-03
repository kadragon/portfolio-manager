"""Tests for Rich-based account add flow."""

from datetime import datetime
from decimal import Decimal
from uuid import uuid4
from unittest.mock import MagicMock

from rich.console import Console

from portfolio_manager.cli.rich_accounts import add_account_flow
from portfolio_manager.models import Account


def test_add_account_flow_creates_account_and_reports_name():
    """Should create account and render confirmation."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.create.return_value = account

    add_account_flow(
        console,
        repo,
        prompt_name=lambda: "Brokerage",
        prompt_cash=lambda: Decimal("1000.25"),
    )

    repo.create.assert_called_once_with(
        name="Brokerage", cash_balance=Decimal("1000.25")
    )
    output = console.export_text()
    assert "Brokerage" in output
