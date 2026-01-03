"""Tests for Rich-based holding delete flow."""

from datetime import datetime
from decimal import Decimal
from uuid import uuid4
from unittest.mock import MagicMock

from rich.console import Console

from portfolio_manager.cli.rich_holdings import delete_holding_flow
from portfolio_manager.models import Account, Holding


def test_delete_holding_flow_removes_holding_and_reports_quantity():
    """Should delete holding and render confirmation."""
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

    delete_holding_flow(console, repo, account, holding, confirm=lambda: True)

    repo.delete.assert_called_once_with(holding.id)
    output = console.export_text()
    assert "5.5" in output
