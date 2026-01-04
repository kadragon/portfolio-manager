"""Tests for Rich-based account flows."""

from datetime import datetime
from decimal import Decimal
from uuid import uuid4
from unittest.mock import MagicMock, patch

from rich.console import Console

from portfolio_manager.cli.accounts import (
    add_account_flow,
    delete_account_flow,
    render_account_list,
    run_account_menu,
)
from portfolio_manager.cli.prompt_select import (
    choose_account_from_list,
    choose_account_menu,
)
from portfolio_manager.cli.app import select_main_menu_option
from portfolio_manager.models import Account


def test_add_account_flow_creates_account_and_reports_name():
    """Should create account and render confirmation."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.create.return_value = account

    add_account_flow(
        console,
        repo,
        prompt_name=lambda: "Brokerage",
        prompt_cash=lambda: Decimal("1000.25"),
    )

    repo.create.assert_called_once_with(
        name="Brokerage", cash_balance=Decimal("1000.25")
    )
    output = console.export_text()
    assert "Brokerage" in output


def test_delete_account_flow_removes_account_and_reports_name():
    """Should delete account and render confirmation."""
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

    delete_account_flow(console, repo, holding_repo, account, confirm=lambda: True)

    repo.delete_with_holdings.assert_called_once_with(account.id, holding_repo)
    output = console.export_text()
    assert "Brokerage" in output


def test_render_account_list_shows_empty_message_when_no_accounts():
    """Should show empty message when no accounts exist."""
    console = Console(record=True, width=80)

    render_account_list(console, [])

    output = console.export_text()
    assert "No accounts found" in output


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
        "portfolio_manager.cli.accounts.choose_account_from_list",
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
        "portfolio_manager.cli.accounts.choose_account_from_list",
        return_value=account.id,
    ):
        with patch(
            "portfolio_manager.cli.accounts.update_account_flow",
        ) as update_account_flow:
            run_account_menu(
                console,
                repo,
                holding_repo,
                prompt=lambda: "b",
                chooser=chooser,
            )

    update_account_flow.assert_called_once()


def test_selecting_accounts_option_returns_accounts_action():
    """Should route to account management when selecting accounts option."""
    action = select_main_menu_option("a")

    assert action == "accounts"


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
