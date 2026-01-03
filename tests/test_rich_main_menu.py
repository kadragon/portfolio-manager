"""Tests for Rich-based main menu rendering."""

from rich.console import Console

from portfolio_manager.cli.rich_app import render_main_menu, select_main_menu_option


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
