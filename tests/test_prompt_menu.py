"""Tests for prompt_toolkit-based menu selection."""

from unittest.mock import MagicMock

from portfolio_manager.cli.prompt_select import choose_main_menu


def test_choose_main_menu_returns_selected_action():
    """Should return the selected action from chooser."""
    chooser = MagicMock(return_value="groups")

    action = choose_main_menu(chooser)

    chooser.assert_called_once()
    assert action == "groups"
