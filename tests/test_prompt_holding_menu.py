"""Tests for prompt_toolkit-based holding menu selection."""

from unittest.mock import MagicMock

from datetime import datetime
from decimal import Decimal
from uuid import uuid4

from portfolio_manager.cli.prompt_select import (
    choose_holding_from_list,
    choose_holding_menu,
)
from portfolio_manager.models import Holding


def test_choose_holding_menu_returns_selected_action():
    """Should return the selected action from chooser."""
    chooser = MagicMock(return_value="delete")

    action = choose_holding_menu(chooser)

    chooser.assert_called_once()
    assert action == "delete"


def test_choose_holding_from_list_returns_holding_id():
    """Should return the selected holding id."""
    holding_id = uuid4()
    holdings = [
        Holding(
            id=holding_id,
            account_id=uuid4(),
            stock_id=uuid4(),
            quantity=Decimal("1.5"),
            created_at=datetime.now(),
            updated_at=datetime.now(),
        ),
    ]
    chooser = MagicMock(return_value=holding_id)

    result = choose_holding_from_list(holdings, chooser)

    chooser.assert_called_once()
    assert result == holding_id


def test_choose_holding_from_list_uses_stock_label_lookup():
    """Should render holding labels using provided stock lookup."""
    holding_id = uuid4()
    stock_id = uuid4()
    holdings = [
        Holding(
            id=holding_id,
            account_id=uuid4(),
            stock_id=stock_id,
            quantity=Decimal("2.0"),
            created_at=datetime.now(),
            updated_at=datetime.now(),
        ),
    ]
    chooser = MagicMock(return_value=holding_id)

    choose_holding_from_list(
        holdings,
        chooser=chooser,
        label_lookup=lambda _stock_id: "AAPL",
    )

    _, kwargs = chooser.call_args
    options = kwargs["options"]
    assert options[0][0] == holding_id
    assert options[0][1] == "AAPL (2.0)"
