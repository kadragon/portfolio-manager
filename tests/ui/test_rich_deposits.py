"""Tests for Rich-based deposit flows."""

from datetime import date, datetime
from decimal import Decimal
from unittest.mock import MagicMock
from uuid import uuid4

from rich.console import Console

from portfolio_manager.cli.deposits import update_deposit_flow
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
