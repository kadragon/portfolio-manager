"""Tests for Rich-based main menu and navigation."""

from unittest.mock import MagicMock, patch

from rich.console import Console

from portfolio_manager.cli.rich_app import (
    handle_main_menu_key,
    render_main_menu,
    select_main_menu_option,
)
from portfolio_manager.cli.rich_menu import render_menu_options
from portfolio_manager.cli.rich_groups import select_group_menu_option
from portfolio_manager.cli import main as main_app
from portfolio_manager.cli.prompt_select import choose_main_menu


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


def test_render_menu_options_prints_options_line():
    """Should print a standardized options line."""
    console = Console(record=True, width=80)

    render_menu_options(console, "g=groups | q=quit")

    output = console.export_text()
    assert "Options:" in output
    assert "g=groups" in output


def test_selecting_back_from_group_menu_returns_back_action():
    """Should return back action when selecting back in group menu."""
    action = select_group_menu_option("b")

    assert action == "back"


def test_main_menu_g_key_returns_groups_action():
    """Should return groups action when pressing 'g'."""
    action = handle_main_menu_key("g")

    assert action == "groups"


def test_choose_main_menu_returns_selected_action():
    """Should return the selected action from chooser."""
    chooser = MagicMock(return_value="groups")

    action = choose_main_menu(chooser)

    chooser.assert_called_once()
    assert action == "groups"
