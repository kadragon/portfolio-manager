"""Tests for prompt_toolkit-based stock menu selection."""

from unittest.mock import MagicMock

from uuid import uuid4

from portfolio_manager.cli.prompt_select import (
    choose_stock_from_list,
    choose_stock_menu,
)
from portfolio_manager.models import Stock


def test_choose_stock_menu_returns_selected_action():
    """Should return the selected action from chooser."""
    chooser = MagicMock(return_value="back")

    action = choose_stock_menu(chooser)

    chooser.assert_called_once()
    assert action == "back"


def test_choose_stock_from_list_returns_stock_id():
    """Should return the selected stock id."""
    stock_id = uuid4()
    stocks = [
        Stock(
            id=stock_id,
            ticker="AAPL",
            group_id=uuid4(),
            created_at=None,  # type: ignore[arg-type]
            updated_at=None,  # type: ignore[arg-type]
        ),
    ]
    chooser = MagicMock(return_value=stock_id)

    result = choose_stock_from_list(stocks, chooser)

    chooser.assert_called_once()
    assert result == stock_id
