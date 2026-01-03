"""Tests for Rich-based navigation actions."""

from portfolio_manager.cli.rich_groups import select_group_menu_option


def test_selecting_back_from_group_menu_returns_back_action():
    """Should return back action when selecting back in group menu."""
    action = select_group_menu_option("b")

    assert action == "back"
