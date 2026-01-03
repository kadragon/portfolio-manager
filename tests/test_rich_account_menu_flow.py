"""Tests for Rich-based account menu flow."""

from datetime import datetime
from decimal import Decimal
from uuid import uuid4
from unittest.mock import MagicMock, patch

from rich.console import Console

from portfolio_manager.cli.rich_accounts import run_account_menu
from portfolio_manager.models import Account


def test_run_account_menu_renders_list_and_allows_back():
    """Should render accounts and exit on back command."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    holding_repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.list_all.return_value = [account]

    chooser = MagicMock(side_effect=["select", "back"])

    with patch(
        "portfolio_manager.cli.rich_accounts.choose_account_from_list",
        return_value=account.id,
    ):
        run_account_menu(
            console,
            repo,
            holding_repo,
            prompt=lambda: "b",
            chooser=chooser,
            holding_chooser=lambda **_: "back",
            holding_prompt=lambda: "b",
        )

    assert repo.list_all.call_count >= 1
    output = console.export_text()
    assert "Current Account" in output
    assert "Brokerage" in output


def test_run_account_menu_edit_flow_invokes_update_account():
    """Should call update flow when selecting edit in account menu."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    holding_repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.list_all.return_value = [account]

    chooser = MagicMock(side_effect=["edit", "back"])

    with patch(
        "portfolio_manager.cli.rich_accounts.choose_account_from_list",
        return_value=account.id,
    ):
        with patch(
            "portfolio_manager.cli.rich_accounts.update_account_flow",
        ) as update_account_flow:
            run_account_menu(
                console,
                repo,
                holding_repo,
                prompt=lambda: "b",
                chooser=chooser,
            )

    update_account_flow.assert_called_once()
