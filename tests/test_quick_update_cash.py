"""Tests for quick update cash balance feature."""

from datetime import datetime
from decimal import Decimal
from unittest.mock import MagicMock
from uuid import uuid4

from rich.console import Console

from portfolio_manager.models import Account
from portfolio_manager.cli.accounts import quick_update_cash_flow
from portfolio_manager.cli.prompt_select import choose_account_menu


def make_account(name: str) -> Account:
    """Create a test account with required fields."""
    now = datetime.now()
    return Account(
        id=uuid4(),
        name=name,
        cash_balance=Decimal("0"),
        created_at=now,
        updated_at=now,
    )


class TestQuickUpdateCashFlow:
    """Tests for quick_update_cash_flow function."""

    def test_updates_all_accounts_in_order(self):
        """Given 3 accounts, prompts for each and updates all."""
        console = Console(force_terminal=True)

        accounts = [
            make_account("Account A"),
            make_account("Account B"),
            make_account("Account C"),
        ]

        # Mock repository
        repository = MagicMock()
        repository.list_all.return_value = accounts

        # Mock prompt returns values in order
        prompt_values = iter([Decimal("1000"), Decimal("2000"), Decimal("3000")])

        def prompt_cash(name: str) -> Decimal:
            return next(prompt_values)

        quick_update_cash_flow(console, repository, prompt_cash=prompt_cash)

        # Verify update was called for each account with correct values
        assert repository.update.call_count == 3

        calls = repository.update.call_args_list
        assert calls[0][1]["name"] == "Account A"
        assert calls[0][1]["cash_balance"] == Decimal("1000")
        assert calls[1][1]["name"] == "Account B"
        assert calls[1][1]["cash_balance"] == Decimal("2000")
        assert calls[2][1]["name"] == "Account C"
        assert calls[2][1]["cash_balance"] == Decimal("3000")

    def test_no_accounts_prints_message(self):
        """When no accounts exist, prints a message and returns."""
        console = Console(force_terminal=True)

        repository = MagicMock()
        repository.list_all.return_value = []

        quick_update_cash_flow(console, repository)

        # No updates should happen
        repository.update.assert_not_called()


class TestAccountMenuHasQuickOption:
    """Tests for account menu quick update option."""

    def test_quick_option_in_menu(self):
        """Account menu should have 'quick' option."""
        selected = []

        def mock_chooser(message, options, default):
            # Capture options for verification
            option_keys = [opt[0] for opt in options]
            selected.extend(option_keys)
            return "quick"

        result = choose_account_menu(chooser=mock_chooser)

        assert "quick" in selected
        assert result == "quick"
