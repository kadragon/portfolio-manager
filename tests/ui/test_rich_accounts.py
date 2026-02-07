"""Tests for Rich-based account flows."""

from datetime import datetime
from decimal import Decimal
from uuid import uuid4
from unittest.mock import MagicMock, patch

from rich.console import Console

from portfolio_manager.cli.accounts import (
    add_account_flow,
    delete_account_flow,
    quick_update_cash_flow,
    render_account_list,
    run_account_menu,
    sync_kis_account_flow,
    update_account_flow,
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


def test_add_account_flow_cancelled_does_not_create():
    """Should not create account when user cancels name input."""
    console = Console(record=True, width=80)
    repo = MagicMock()

    add_account_flow(
        console,
        repo,
        prompt_name=lambda: None,
        prompt_cash=lambda: Decimal("1000.25"),
    )

    repo.create.assert_not_called()
    output = console.export_text()
    assert "Cancelled" in output


def test_update_account_flow_blank_inputs_keep_existing_values():
    """Blank account name/cash inputs should keep existing values."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Main",
        cash_balance=Decimal("1500.50"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.update.return_value = account

    update_account_flow(
        console,
        repo,
        account,
        prompt_name=lambda: "   ",
        prompt_cash=lambda: "",
    )

    repo.update.assert_called_once_with(
        account.id,
        name="Main",
        cash_balance=Decimal("1500.50"),
    )


def test_update_account_flow_invalid_cash_keeps_existing_value():
    """Invalid cash input should keep existing cash balance."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Main",
        cash_balance=Decimal("1500.50"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.update.return_value = account

    update_account_flow(
        console,
        repo,
        account,
        prompt_name=lambda: "Renamed",
        prompt_cash=lambda: "abc",
    )

    repo.update.assert_called_once_with(
        account.id,
        name="Renamed",
        cash_balance=Decimal("1500.50"),
    )
    output = console.export_text()
    assert "Invalid cash balance" in output


def test_quick_update_cash_flow_blank_input_keeps_existing_balance():
    """Blank quick-update input should keep each account's current balance."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    account_a = Account(
        id=uuid4(),
        name="Main",
        cash_balance=Decimal("1000"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    account_b = Account(
        id=uuid4(),
        name="Sub",
        cash_balance=Decimal("2000"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.list_all.return_value = [account_a, account_b]

    prompt_values = iter([" ", "3000"])

    def prompt_cash(_name: str):
        return next(prompt_values)

    quick_update_cash_flow(console, repo, prompt_cash=prompt_cash)

    calls = repo.update.call_args_list
    assert calls[0][1]["name"] == "Main"
    assert calls[0][1]["cash_balance"] == Decimal("1000")
    assert calls[1][1]["name"] == "Sub"
    assert calls[1][1]["cash_balance"] == Decimal("3000")


def test_sync_kis_account_flow_invokes_sync_service():
    """Should call KIS sync service and report completion."""
    console = Console(record=True, width=80)
    account = Account(
        id=uuid4(),
        name="한국투자증권",
        cash_balance=Decimal("0"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    sync_service = MagicMock()
    sync_service.sync_account.return_value = MagicMock(
        cash_balance=Decimal("500000"),
        holding_count=2,
        created_stock_count=1,
    )

    sync_kis_account_flow(
        console,
        account,
        sync_service,
        cano="12345678",
        acnt_prdt_cd="01",
    )

    sync_service.sync_account.assert_called_once_with(
        account=account,
        cano="12345678",
        acnt_prdt_cd="01",
    )
    output = console.export_text()
    assert "KIS synced" in output


def test_run_account_menu_sync_flow_uses_selected_account():
    """Should route sync action to KIS sync flow for selected account."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    holding_repo = MagicMock()
    stock_repo = MagicMock()
    group_repo = MagicMock()
    sync_service = MagicMock()
    account = Account(
        id=uuid4(),
        name="한국투자증권",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.list_all.return_value = [account]

    chooser = MagicMock(side_effect=["sync", "back"])

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
            stock_repository=stock_repo,
            group_repository=group_repo,
            kis_sync_service=sync_service,
            kis_cano="12345678",
            kis_acnt_prdt_cd="01",
        )

    sync_service.sync_account.assert_called_once_with(
        account=account,
        cano="12345678",
        acnt_prdt_cd="01",
    )
