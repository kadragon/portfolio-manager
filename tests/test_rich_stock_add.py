"""Tests for Rich-based stock add flow."""

from datetime import datetime
from uuid import uuid4
from unittest.mock import MagicMock

from rich.console import Console

from portfolio_manager.cli.rich_stocks import add_stock_flow
from portfolio_manager.models import Group, Stock


def test_add_stock_flow_creates_stock_and_reports_ticker():
    """Should create stock and render confirmation."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    group = Group(
        id=uuid4(),
        name="Tech Stocks",
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    stock = Stock(
        id=uuid4(),
        ticker="AAPL",
        group_id=group.id,
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.create.return_value = stock

    add_stock_flow(console, repo, group, prompt=lambda: "AAPL")

    repo.create.assert_called_once_with("AAPL", group.id)
    output = console.export_text()
    assert "AAPL" in output
