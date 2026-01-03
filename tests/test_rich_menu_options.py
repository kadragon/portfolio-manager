"""Tests for shared menu option rendering."""

from rich.console import Console

from portfolio_manager.cli.rich_menu import render_menu_options


def test_render_menu_options_prints_options_line():
    """Should print a standardized options line."""
    console = Console(record=True, width=80)

    render_menu_options(console, "g=groups | q=quit")

    output = console.export_text()
    assert "Options:" in output
    assert "g=groups" in output
