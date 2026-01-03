"""Tests for prompt_toolkit-based account menu selection."""

from unittest.mock import MagicMock

from datetime import datetime
from decimal import Decimal
from uuid import uuid4

from portfolio_manager.cli.prompt_select import (
    choose_account_from_list,
    choose_account_menu,
)
from portfolio_manager.models import Account


def test_choose_account_menu_returns_selected_action():
    """Should return the selected action from chooser."""
    chooser = MagicMock(return_value="select")

    action = choose_account_menu(chooser)

    chooser.assert_called_once()
    assert action == "select"


def test_choose_account_from_list_returns_account_id():
    """Should return the selected account id."""
    account_id = uuid4()
    accounts = [
        Account(
            id=account_id,
            name="Brokerage",
            cash_balance=Decimal("10.00"),
            created_at=datetime.now(),
            updated_at=datetime.now(),
        ),
    ]
    chooser = MagicMock(return_value=account_id)

    result = choose_account_from_list(accounts, chooser)

    chooser.assert_called_once()
    assert result == account_id
