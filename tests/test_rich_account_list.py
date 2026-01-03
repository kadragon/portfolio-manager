"""Tests for Rich-based account list rendering."""

from rich.console import Console

from portfolio_manager.cli.rich_accounts import render_account_list


def test_render_account_list_shows_empty_message_when_no_accounts():
    """Should show empty message when no accounts exist."""
    console = Console(record=True, width=80)

    render_account_list(console, [])

    output = console.export_text()
    assert "No accounts found" in output
