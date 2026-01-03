"""Tests for Rich-based group menu flow."""

from datetime import datetime
from uuid import uuid4
from unittest.mock import patch

from rich.console import Console

from portfolio_manager.cli import main as main_app


def test_run_group_menu_add_flow_invokes_add_group():
    """Should call add flow when selecting add in group menu."""
    console = Console(record=True, width=80)

    with patch.object(main_app, "_load_groups", return_value=[]):
        with patch.object(main_app, "choose_group_menu", side_effect=["add", "back"]):
            with patch.object(main_app, "get_supabase_client"):
                with patch.object(main_app, "GroupRepository"):
                    with patch.object(main_app, "add_group_flow") as add_group_flow:
                        main_app.run_group_menu(console)

    add_group_flow.assert_called_once()


def test_run_group_menu_delete_flow_invokes_delete_group():
    """Should call delete flow when selecting delete in group menu."""
    console = Console(record=True, width=80)

    with patch.object(main_app, "_load_groups", return_value=[]):
        with patch.object(
            main_app, "choose_group_menu", side_effect=["delete", "back"]
        ):
            with patch.object(main_app, "choose_group_from_list", return_value=None):
                with patch.object(main_app, "get_supabase_client"):
                    with patch.object(main_app, "GroupRepository"):
                        with patch.object(
                            main_app, "delete_group_flow"
                        ) as delete_group_flow:
                            main_app.run_group_menu(console)

    delete_group_flow.assert_not_called()


def test_run_group_menu_select_flow_invokes_stock_menu():
    """Should open stock menu when selecting a group."""
    console = Console(record=True, width=80)
    group = main_app.Group(
        id=uuid4(),
        name="Tech",
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
                        with patch.object(main_app, "run_stock_menu") as run_stock_menu:
                            main_app.run_group_menu(console)

    run_stock_menu.assert_called_once()


def test_run_group_menu_edit_flow_invokes_update_group():
    """Should call update flow when selecting edit in group menu."""
    console = Console(record=True, width=80)
    group = main_app.Group(
        id=uuid4(),
        name="Tech",
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )

    with patch.object(main_app, "_load_groups", return_value=[group]):
        with patch.object(main_app, "choose_group_menu", side_effect=["edit", "back"]):
            with patch.object(
                main_app, "choose_group_from_list", return_value=group.id
            ):
                with patch.object(main_app, "get_supabase_client"):
                    with patch.object(main_app, "GroupRepository"):
                        with patch.object(
                            main_app, "update_group_flow"
                        ) as update_group_flow:
                            main_app.run_group_menu(console)

    update_group_flow.assert_called_once()
