"""Tests for Rich-based keyboard command handling."""

from portfolio_manager.cli.rich_app import handle_main_menu_key


def test_main_menu_g_key_returns_groups_action():
    """Should return groups action when pressing 'g'."""
    action = handle_main_menu_key("g")

    assert action == "groups"
