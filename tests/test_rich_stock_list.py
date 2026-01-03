"""Tests for Rich-based stock list rendering."""

from datetime import datetime
from uuid import uuid4
from unittest.mock import MagicMock

from rich.console import Console

from portfolio_manager.cli.rich_stocks import render_stocks_for_group
from portfolio_manager.models import Group, Stock


def test_render_stocks_for_group_lists_stocks():
    """Should render stocks for the selected group."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    group = Group(
        id=uuid4(),
        name="Tech Stocks",
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    stocks = [
        Stock(
            id=uuid4(),
            ticker="AAPL",
            group_id=group.id,
            created_at=datetime.now(),
            updated_at=datetime.now(),
        ),
        Stock(
            id=uuid4(),
            ticker="GOOGL",
            group_id=group.id,
            created_at=datetime.now(),
            updated_at=datetime.now(),
        ),
    ]
    repo.list_by_group.return_value = stocks

    render_stocks_for_group(console, repo, group)

    repo.list_by_group.assert_called_once_with(group.id)
    output = console.export_text()
    assert "AAPL" in output
    assert "GOOGL" in output
