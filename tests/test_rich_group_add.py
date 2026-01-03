"""Tests for Rich-based group add flow."""

from datetime import datetime
from uuid import uuid4
from unittest.mock import MagicMock

from rich.console import Console

from portfolio_manager.cli.rich_groups import add_group_flow
from portfolio_manager.models import Group


def test_add_group_flow_creates_group_and_reports_name():
    """Should create a group and render confirmation."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    group = Group(
        id=uuid4(),
        name="Tech Stocks",
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.create.return_value = group

    add_group_flow(console, repo, lambda: "Tech Stocks")

    repo.create.assert_called_once_with("Tech Stocks")
    output = console.export_text()
    assert "Tech Stocks" in output
