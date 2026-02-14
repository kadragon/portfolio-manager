"""Tests for cancellable prompt functionality."""

from datetime import date, datetime
from decimal import Decimal
from unittest.mock import MagicMock
from uuid import uuid4

import pytest
from rich.console import Console

from portfolio_manager.models import Account, Deposit, Group, Holding, Stock


class TestCancellablePrompt:
    """Test cancellable_prompt returns None on ESC or Ctrl+C."""

    def test_returns_input_on_normal_entry(self):
        """Normal input should return the entered value."""
        from portfolio_manager.cli.prompt_select import cancellable_prompt

        mock_session = MagicMock()
        mock_session.prompt.return_value = "test value"

        result = cancellable_prompt("Enter name:", session=mock_session)

        assert result == "test value"
        mock_session.prompt.assert_called_once()

    def test_returns_none_on_keyboard_interrupt(self):
        """Ctrl+C should return None instead of raising."""
        from portfolio_manager.cli.prompt_select import cancellable_prompt

        mock_session = MagicMock()
        mock_session.prompt.side_effect = KeyboardInterrupt()

        result = cancellable_prompt("Enter name:", session=mock_session)

        assert result is None

    def test_returns_none_on_eof_error(self):
        """Ctrl+D or ESC abort should return None."""
        from portfolio_manager.cli.prompt_select import cancellable_prompt

        mock_session = MagicMock()
        mock_session.prompt.side_effect = EOFError()

        result = cancellable_prompt("Enter name:", session=mock_session)

        assert result is None

    def test_accepts_default_value(self):
        """Default value should be passed to the prompt."""
        from portfolio_manager.cli.prompt_select import cancellable_prompt

        mock_session = MagicMock()
        mock_session.prompt.return_value = "default_val"

        result = cancellable_prompt(
            "Enter name:", default="default_val", session=mock_session
        )

        assert result == "default_val"
        call_kwargs = mock_session.prompt.call_args
        assert call_kwargs[1].get("default") == "default_val"

    def test_prompt_decimal_retries_until_valid_decimal(self):
        """prompt_decimal should retry on invalid input and return Decimal."""
        from portfolio_manager.cli.prompt_select import prompt_decimal

        mock_session = MagicMock()
        mock_session.prompt.side_effect = ["abc", "12.50"]

        result = prompt_decimal("Amount:", session=mock_session)

        assert result == Decimal("12.50")

    def test_prompt_decimal_shows_error_message_on_invalid_input(self):
        """Invalid decimal input should render an error message before retrying."""
        from portfolio_manager.cli.prompt_select import prompt_decimal

        mock_session = MagicMock()
        mock_session.prompt.side_effect = ["abc", "12.50"]
        console = Console(record=True, width=80)

        result = prompt_decimal("Amount:", session=mock_session, console=console)

        assert result == Decimal("12.50")
        output = console.export_text()
        assert "Invalid number. Please try again." in output

    def test_prompt_decimal_returns_none_when_cancelled(self):
        """prompt_decimal should return None when prompt is cancelled."""
        from portfolio_manager.cli.prompt_select import prompt_decimal

        mock_session = MagicMock()
        mock_session.prompt.side_effect = KeyboardInterrupt()

        result = prompt_decimal("Amount:", session=mock_session)

        assert result is None


def test_create_esc_bindings_exits_with_eof_error():
    """ESC key binding should exit prompt app with EOFError."""
    from portfolio_manager.cli.prompt_select import _create_esc_bindings

    bindings = _create_esc_bindings()
    event = MagicMock()

    bindings.bindings[0].handler(event)

    exception = event.app.exit.call_args.kwargs["exception"]
    assert isinstance(exception, EOFError)


@pytest.mark.parametrize(
    ("func_name", "expected"),
    [
        ("choose_main_menu", "groups"),
        ("choose_group_menu", "add"),
        ("choose_stock_menu", "add"),
        ("choose_account_menu", "quick"),
        ("choose_holding_menu", "add"),
        ("choose_deposit_menu", "add"),
        ("choose_rebalance_action", "preview"),
    ],
)
def test_choose_menu_functions_use_default_choice_import(func_name: str, expected: str):
    """Menu pickers should import and use prompt_toolkit.choice when chooser is None."""
    from portfolio_manager.cli import prompt_select

    with pytest.MonkeyPatch.context() as mp:
        fake_choice = MagicMock(return_value=expected)
        mp.setattr("prompt_toolkit.shortcuts.choice", fake_choice)
        result = getattr(prompt_select, func_name)()

    assert result == expected
    fake_choice.assert_called_once()


def test_choose_group_from_list_returns_none_for_empty_options():
    """Group chooser should return None when list is empty."""
    from portfolio_manager.cli.prompt_select import choose_group_from_list

    with pytest.MonkeyPatch.context() as mp:
        mp.setattr("prompt_toolkit.shortcuts.choice", MagicMock())
        assert choose_group_from_list([]) is None


def test_choose_account_from_list_returns_none_for_empty_options():
    """Account chooser should return None when list is empty."""
    from portfolio_manager.cli.prompt_select import choose_account_from_list

    with pytest.MonkeyPatch.context() as mp:
        mp.setattr("prompt_toolkit.shortcuts.choice", MagicMock())
        assert choose_account_from_list([]) is None


def test_choose_stock_from_list_returns_none_for_empty_options():
    """Stock chooser should return None when list is empty."""
    from portfolio_manager.cli.prompt_select import choose_stock_from_list

    with pytest.MonkeyPatch.context() as mp:
        mp.setattr("prompt_toolkit.shortcuts.choice", MagicMock())
        assert choose_stock_from_list([]) is None


def test_choose_holding_from_list_returns_none_for_empty_options():
    """Holding chooser should return None when list is empty."""
    from portfolio_manager.cli.prompt_select import choose_holding_from_list

    with pytest.MonkeyPatch.context() as mp:
        mp.setattr("prompt_toolkit.shortcuts.choice", MagicMock())
        assert choose_holding_from_list([]) is None


def test_choose_deposit_from_list_returns_none_for_empty_options():
    """Deposit chooser should return None when list is empty."""
    from portfolio_manager.cli.prompt_select import choose_deposit_from_list

    with pytest.MonkeyPatch.context() as mp:
        mp.setattr("prompt_toolkit.shortcuts.choice", MagicMock())
        assert choose_deposit_from_list([]) is None


def test_prompt_decimal_invalid_without_console_prints_to_stdout(capsys):
    """Invalid decimal should print fallback message when console is omitted."""
    from portfolio_manager.cli.prompt_select import prompt_decimal

    mock_session = MagicMock()
    mock_session.prompt.side_effect = ["invalid", "1.23"]

    result = prompt_decimal("Amount:", session=mock_session, console=None)

    assert result == Decimal("1.23")
    output = capsys.readouterr().out
    assert "Invalid number. Please try again." in output


def test_choose_list_functions_use_default_choice_import_when_non_empty():
    """List pickers should use default imported choice with non-empty data."""
    from portfolio_manager.cli.prompt_select import (
        choose_account_from_list,
        choose_deposit_from_list,
        choose_group_from_list,
        choose_holding_from_list,
        choose_stock_from_list,
    )

    now = datetime.now()
    group_id = uuid4()
    account_id = uuid4()
    stock_id = uuid4()
    holding_id = uuid4()
    deposit_id = uuid4()

    groups = [Group(id=group_id, name="G1", created_at=now, updated_at=now)]
    accounts = [
        Account(
            id=account_id,
            name="A1",
            cash_balance=Decimal("1"),
            created_at=now,
            updated_at=now,
        )
    ]
    stocks = [
        Stock(
            id=stock_id,
            ticker="AAPL",
            group_id=group_id,
            created_at=now,
            updated_at=now,
        )
    ]
    holdings = [
        Holding(
            id=holding_id,
            account_id=account_id,
            stock_id=stock_id,
            quantity=Decimal("2"),
            created_at=now,
            updated_at=now,
        )
    ]
    deposits = [
        Deposit(
            id=deposit_id,
            amount=Decimal("100"),
            deposit_date=date(2026, 2, 6),
            note=None,
            created_at=now,
            updated_at=now,
        )
    ]

    with pytest.MonkeyPatch.context() as mp:
        fake_choice = MagicMock(
            side_effect=[group_id, account_id, stock_id, holding_id, deposit_id]
        )
        mp.setattr("prompt_toolkit.shortcuts.choice", fake_choice)
        assert choose_group_from_list(groups) == group_id
        assert choose_account_from_list(accounts) == account_id
        assert choose_stock_from_list(stocks) == stock_id
        assert choose_holding_from_list(holdings) == holding_id
        assert choose_deposit_from_list(deposits) == deposit_id

    assert fake_choice.call_count == 5
