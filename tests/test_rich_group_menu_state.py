"""Tests for group menu selection state display."""

from datetime import datetime
from unittest.mock import patch
from uuid import uuid4

from rich.console import Console

from portfolio_manager.models import Group
from portfolio_manager.cli import main as main_app


def test_run_group_menu_displays_current_group_after_selection():
    """Should display current group after a selection."""
    console = Console(record=True, width=80)
    group = Group(
        id=uuid4(),
        name="Tech Stocks",
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )

    with patch.object(main_app, "_load_groups", return_value=[group]):
        with patch.object(
            main_app, "choose_group_menu", side_effect=["select", "back"]
        ):
            with patch.object(
                main_app, "choose_group_from_list", return_value=group.id
            ):
                with patch.object(main_app, "get_supabase_client"):
                    with patch.object(main_app, "StockRepository"):
                        with patch.object(main_app, "run_stock_menu"):
                            main_app.run_group_menu(console)

    output = console.export_text()
    assert "Current Group" in output
    assert "Tech Stocks" in output
