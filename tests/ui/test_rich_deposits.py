"""Tests for Rich-based deposit flows."""

from datetime import date, datetime
from decimal import Decimal
from unittest.mock import MagicMock, patch
from uuid import uuid4

from rich.console import Console

from portfolio_manager.cli.deposits import (
    delete_deposit_flow,
    get_date_input,
    select_deposit_to_edit,
    update_deposit_flow,
)
from portfolio_manager.cli.prompt_select import choose_deposit_from_list
from portfolio_manager.models import Deposit


def _make_deposit(*, amount: Decimal, note: str | None = None) -> Deposit:
    now = datetime.now()
    return Deposit(
        id=uuid4(),
        amount=amount,
        deposit_date=date(2026, 2, 6),
        note=note,
        created_at=now,
        updated_at=now,
    )


def test_update_deposit_flow_blank_amount_keeps_existing_value():
    """Blank amount input should keep the current amount."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    deposit = _make_deposit(amount=Decimal("100000"), note="seed")

    update_deposit_flow(
        console,
        repo,
        deposit,
        prompt_amount=lambda: "   ",
        prompt_note=lambda: "updated",
    )

    repo.update.assert_called_once_with(
        deposit.id,
        amount=Decimal("100000"),
        note="updated",
    )


def test_update_deposit_flow_blank_note_keeps_existing_value():
    """Blank note input should keep the current note."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    deposit = _make_deposit(amount=Decimal("100000"), note="existing note")

    update_deposit_flow(
        console,
        repo,
        deposit,
        prompt_amount=lambda: "100001",
        prompt_note=lambda: "   ",
    )

    repo.update.assert_called_once_with(
        deposit.id,
        amount=Decimal("100001"),
    )


def test_update_deposit_flow_clear_note_with_clear_keyword():
    """/clear should explicitly clear note to NULL."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    deposit = _make_deposit(amount=Decimal("100000"), note="existing note")

    update_deposit_flow(
        console,
        repo,
        deposit,
        prompt_amount=lambda: "100001",
        prompt_note=lambda: " /ClEaR ",
    )

    repo.update.assert_called_once_with(
        deposit.id,
        amount=Decimal("100001"),
        note=None,
    )


def test_update_deposit_flow_invalid_amount_keeps_existing_value():
    """Invalid amount input should keep current amount."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    deposit = _make_deposit(amount=Decimal("100000"), note="seed")

    update_deposit_flow(
        console,
        repo,
        deposit,
        prompt_amount=lambda: "abc",
        prompt_note=lambda: "updated",
    )

    repo.update.assert_called_once_with(
        deposit.id,
        amount=Decimal("100000"),
        note="updated",
    )
    output = console.export_text()
    assert "Invalid amount" in output


def test_choose_deposit_from_list_returns_deposit_id():
    """Should return selected deposit id from chooser."""
    deposit = _make_deposit(amount=Decimal("1000"))
    chooser = MagicMock(return_value=deposit.id)

    result = choose_deposit_from_list([deposit], chooser)

    chooser.assert_called_once()
    assert result == deposit.id


def test_delete_deposit_flow_deletes_selected_deposit_with_choice_picker():
    """Delete flow should use choice-based selection and delete chosen deposit."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    deposit = _make_deposit(amount=Decimal("1000"))

    with patch(
        "portfolio_manager.cli.deposits.choose_deposit_from_list",
        return_value=deposit.id,
    ):
        with patch("portfolio_manager.cli.deposits.Confirm.ask", return_value=True):
            delete_deposit_flow(console, repo, [deposit])

    repo.delete.assert_called_once_with(deposit.id)


def test_select_deposit_to_edit_updates_selected_deposit_with_choice_picker():
    """Edit selection should use choice picker and update chosen deposit."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    deposit = _make_deposit(amount=Decimal("1000"))

    with patch(
        "portfolio_manager.cli.deposits.choose_deposit_from_list",
        return_value=deposit.id,
    ):
        with patch("portfolio_manager.cli.deposits.update_deposit_flow") as mock_update:
            select_deposit_to_edit(console, repo, [deposit])

    mock_update.assert_called_once_with(console, repo, deposit)


def test_get_date_input_renders_invalid_format_with_console():
    """Invalid date format should render a console warning and retry."""
    console = Console(record=True, width=80)
    responses = iter(["2026/02/06", "2026-02-06"])

    result = get_date_input(
        prompt_func=lambda: next(responses),
        console=console,
    )

    assert result == date(2026, 2, 6)
    output = console.export_text()
    assert "Invalid format" in output
