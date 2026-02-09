"""Rich-based account list rendering."""

from decimal import Decimal, InvalidOperation
from typing import Callable

from rich.console import Console
from rich import box
from rich.panel import Panel
from rich.prompt import Confirm
from rich.table import Table

from portfolio_manager.models import Account
from portfolio_manager.cli.holdings import run_holdings_menu
from portfolio_manager.cli.prompt_select import (
    cancellable_prompt,
    choose_account_from_list,
    choose_account_menu,
    prompt_decimal,
)


def sync_kis_account_flow(
    console: Console,
    account: Account,
    kis_sync_service,
    *,
    cano: str,
    acnt_prdt_cd: str,
) -> None:
    """Sync one account from KIS API and render status."""
    try:
        result = kis_sync_service.sync_account(
            account=account,
            cano=cano,
            acnt_prdt_cd=acnt_prdt_cd,
        )
    except Exception as exc:
        console.print(f"[red]KIS sync failed: {exc}[/red]")
        return

    console.print(
        f"KIS synced {account.name}: "
        f"cash={result.cash_balance}, "
        f"holdings={result.holding_count}, "
        f"new_stocks={result.created_stock_count}"
    )


def _resolve_decimal_input(
    raw_value: Decimal | str,
    current_value: Decimal,
    *,
    on_invalid: Callable[[], None] | None = None,
) -> Decimal:
    """Resolve prompt input into Decimal, retaining current value for blank/invalid."""
    if isinstance(raw_value, Decimal):
        return raw_value

    value_text = raw_value.strip()
    if value_text == "":
        return current_value

    try:
        return Decimal(value_text)
    except InvalidOperation:
        if on_invalid is not None:
            on_invalid()
        return current_value


def render_account_list(console: Console, accounts: list[Account]) -> None:
    """Render the account list or an empty-state message."""
    if not accounts:
        console.print("No accounts found. Add an account to continue.")
        return

    table = Table(title="Accounts", header_style="bold")
    table.add_column("#", style="dim", width=4, justify="right")
    table.add_column("Name", style="bold")
    table.add_column("Cash Balance", justify="right")
    for index, account in enumerate(accounts, start=1):
        table.add_row(str(index), account.name, str(account.cash_balance))
    console.print(table)


def add_account_flow(
    console: Console,
    repository,
    prompt_name: Callable[[], str | None] | None = None,
    prompt_cash: Callable[[], Decimal | None] | None = None,
) -> None:
    """Add an account via prompts and render confirmation."""
    name_func = prompt_name or (lambda: cancellable_prompt("Account name:"))
    cash_func = prompt_cash or (
        lambda: prompt_decimal("Cash balance:", console=console)
    )
    name = name_func()
    if name is None:
        console.print("[yellow]Cancelled[/yellow]")
        return
    cash = cash_func()
    if cash is None:
        console.print("[yellow]Cancelled[/yellow]")
        return
    account = repository.create(name=name, cash_balance=cash)
    console.print(f"Added account: {account.name}")


def update_account_flow(
    console: Console,
    repository,
    account: Account,
    prompt_name: Callable[[], str | None] | None = None,
    prompt_cash: Callable[[], Decimal | str | None] | None = None,
) -> None:
    """Update an account via prompts and render confirmation."""
    name_func = prompt_name or (
        lambda: cancellable_prompt("New account name:", default=account.name)
    )
    cash_func = prompt_cash or (
        lambda: cancellable_prompt(
            "New cash balance:", default=str(account.cash_balance)
        )
    )
    name_input = name_func()
    if name_input is None:
        console.print("[yellow]Cancelled[/yellow]")
        return
    cash_input = cash_func()
    if cash_input is None:
        console.print("[yellow]Cancelled[/yellow]")
        return

    name = account.name if name_input.strip() == "" else name_input

    cash = _resolve_decimal_input(
        cash_input,
        account.cash_balance,
        on_invalid=lambda: console.print(
            "[yellow]Invalid cash balance, keeping current value[/yellow]"
        ),
    )

    updated = repository.update(
        account.id,
        name=name,
        cash_balance=cash,
    )
    console.print(f"Updated account: {updated.name}")


def delete_account_flow(
    console: Console,
    repository,
    holding_repository,
    account: Account,
    confirm: Callable[[], bool] | None = None,
) -> None:
    """Delete an account with confirmation and render status."""
    confirm_func = confirm or (
        lambda: Confirm.ask(f"Delete {account.name}?", default=False)
    )
    if not confirm_func():
        return
    repository.delete_with_holdings(account.id, holding_repository)
    console.print(f"Deleted account: {account.name}")


def quick_update_cash_flow(
    console: Console,
    repository,
    prompt_cash: Callable[[str], Decimal | str | None] | None = None,
) -> None:
    """Update cash balance for all accounts in sequence."""
    accounts = repository.list_all()
    if not accounts:
        console.print("No accounts to update")
        return

    def _default_cash(name: str, current: Decimal) -> str | None:
        return cancellable_prompt(f"Cash balance for {name}:", default=str(current))

    for account in accounts:
        raw_balance = (
            prompt_cash(account.name)
            if prompt_cash is not None
            else _default_cash(account.name, account.cash_balance)
        )
        if raw_balance is None:
            console.print("[yellow]Cancelled[/yellow]")
            return
        new_balance = _resolve_decimal_input(
            raw_balance,
            account.cash_balance,
            on_invalid=lambda: console.print(
                f"[yellow]Invalid cash balance for {account.name}, keeping current value[/yellow]"
            ),
        )
        repository.update(account.id, name=account.name, cash_balance=new_balance)
        console.print(f"Updated {account.name}: {new_balance}")


def _select_account_by_id(accounts: list[Account], account_id) -> Account | None:
    for account in accounts:
        if account.id == account_id:
            return account
    return None


def run_account_menu(
    console: Console,
    repository,
    holding_repository,
    prompt: Callable[[], str],
    holding_prompt: Callable[[], str] | None = None,
    stock_repository=None,
    chooser: Callable | None = None,
    holding_chooser: Callable | None = None,
    group_repository=None,
    group_chooser: Callable | None = None,
    kis_sync_service=None,
    kis_cano: str | None = None,
    kis_acnt_prdt_cd: str | None = None,
) -> None:
    """Run the account management menu loop."""
    selected_account: Account | None = None
    holding_prompt_func = holding_prompt or (
        lambda: cancellable_prompt("Holdings menu:")
    )
    while True:
        accounts = repository.list_all()
        render_account_list(console, accounts)
        if selected_account is not None:
            console.print(
                Panel.fit(
                    f"[bold]{selected_account.name}[/bold]",
                    title="💼 Current Account",
                    border_style="green",
                    box=box.ROUNDED,
                    padding=(0, 2),
                )
            )
        action = choose_account_menu(chooser)
        if action == "back":
            return
        if action == "quick":
            quick_update_cash_flow(console, repository)
            continue
        if action == "sync":
            if kis_sync_service is None or not kis_cano or not kis_acnt_prdt_cd:
                console.print(
                    "[yellow]KIS sync is not configured. "
                    "Set KIS_CANO and KIS_ACNT_PRDT_CD in .env.[/yellow]"
                )
                continue
            account_id = choose_account_from_list(accounts)
            if account_id is not None:
                account = _select_account_by_id(accounts, account_id)
                if account is not None:
                    sync_kis_account_flow(
                        console,
                        account,
                        kis_sync_service,
                        cano=kis_cano,
                        acnt_prdt_cd=kis_acnt_prdt_cd,
                    )
            continue
        if action == "add":
            add_account_flow(console, repository)
            continue
        if action == "edit":
            account_id = choose_account_from_list(accounts)
            if account_id is not None:
                account = _select_account_by_id(accounts, account_id)
                if account is not None:
                    update_account_flow(console, repository, account)
            continue
        if action == "delete":
            account_id = choose_account_from_list(accounts)
            if account_id is not None:
                account = _select_account_by_id(accounts, account_id)
                if account is not None:
                    delete_account_flow(
                        console, repository, holding_repository, account
                    )
            continue
        if action == "select":
            account_id = choose_account_from_list(accounts)
            if account_id is not None:
                account = _select_account_by_id(accounts, account_id)
                if account is not None:
                    selected_account = account
                    run_holdings_menu(
                        console,
                        holding_repository,
                        account,
                        prompt=holding_prompt_func,
                        stock_repository=stock_repository,
                        chooser=holding_chooser,
                        group_repository=group_repository,
                        group_chooser=group_chooser,
                    )
            continue
