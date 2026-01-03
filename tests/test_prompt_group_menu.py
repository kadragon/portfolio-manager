"""Tests for prompt_toolkit-based group menu selection."""

from unittest.mock import MagicMock

from uuid import uuid4

from portfolio_manager.cli.prompt_select import (
    choose_group_from_list,
    choose_group_menu,
)
from portfolio_manager.models import Group


def test_choose_group_menu_returns_selected_action():
    """Should return the selected action from chooser."""
    chooser = MagicMock(return_value="add")

    action = choose_group_menu(chooser)

    chooser.assert_called_once()
    assert action == "add"


def test_choose_group_from_list_returns_group_id():
    """Should return the selected group id."""
    group_id = uuid4()
    groups = [
        Group(id=group_id, name="Tech", created_at=None, updated_at=None),  # type: ignore[arg-type]
    ]
    chooser = MagicMock(return_value=group_id)

    result = choose_group_from_list(groups, chooser)

    chooser.assert_called_once()
    assert result == group_id
