"""Tests for Rich account menu navigation."""

from portfolio_manager.cli.rich_app import select_main_menu_option


def test_selecting_accounts_option_returns_accounts_action():
    """Should route to account management when selecting accounts option."""
    action = select_main_menu_option("a")

    assert action == "accounts"
