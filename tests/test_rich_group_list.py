"""Tests for Rich-based group list rendering."""

from rich.console import Console

from portfolio_manager.cli.rich_groups import render_group_list


def test_render_group_list_shows_empty_message_when_no_groups():
    """Should show empty message when no groups exist."""
    console = Console(record=True, width=80)

    render_group_list(console, [])

    output = console.export_text()
    assert "No groups found" in output
