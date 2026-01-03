"""Tests for Rich-based group delete flow."""

from datetime import datetime
from uuid import uuid4
from unittest.mock import MagicMock

from rich.console import Console

from portfolio_manager.cli.rich_groups import delete_group_flow
from portfolio_manager.models import Group


def test_delete_group_flow_removes_group_and_reports_name():
    """Should delete group and render confirmation."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    group = Group(
        id=uuid4(),
        name="Tech Stocks",
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )

    delete_group_flow(console, repo, group, confirm=lambda: True)

    repo.delete.assert_called_once_with(group.id)
    output = console.export_text()
    assert "Tech Stocks" in output
