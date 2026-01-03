"""Tests for Rich-based stock menu flow."""

from datetime import datetime
from uuid import uuid4
from unittest.mock import MagicMock

from rich.console import Console

from portfolio_manager.cli.rich_stocks import run_stock_menu
from portfolio_manager.models import Group


def test_run_stock_menu_renders_list_and_allows_back():
    """Should render stocks and exit on back command."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    group = Group(
        id=uuid4(),
        name="Tech Stocks",
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.list_by_group.return_value = []

    run_stock_menu(console, repo, group, prompt=lambda: "b")

    repo.list_by_group.assert_called_once_with(group.id)
