"""Tests for Rich-based main menu rendering."""

from unittest.mock import patch

from rich.console import Console

from portfolio_manager.cli.rich_app import render_main_menu, select_main_menu_option
from portfolio_manager.cli import main as main_app


def test_rich_main_menu_renders_title():
    """Should render the main menu title using Rich."""
    console = Console(record=True, width=80)

    render_main_menu(console)

    output = console.export_text()
    assert "Portfolio Manager" in output


def test_selecting_groups_option_returns_groups_action():
    """Should route to group management when selecting groups option."""
    action = select_main_menu_option("g")

    assert action == "groups"


def test_main_menu_quit_exits():
    """Should exit the main loop when selecting quit."""
    with patch.object(main_app, "render_main_menu") as render_menu:
        with patch.object(main_app, "choose_main_menu", return_value="quit"):
            with patch.object(main_app, "run_group_menu") as run_group_menu:
                with patch.object(main_app, "run_account_menu") as run_account_menu:
                    main_app.main()

    render_menu.assert_called()
    run_group_menu.assert_not_called()
    run_account_menu.assert_not_called()
