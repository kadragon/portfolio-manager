"""Rich-based account list rendering."""

from decimal import Decimal
from typing import Callable

from rich.console import Console
from rich import box
from rich.panel import Panel
from rich.prompt import Confirm, Prompt
from rich.table import Table

from portfolio_manager.models import Account
from portfolio_manager.cli.rich_holdings import run_holdings_menu
from portfolio_manager.cli.prompt_select import (
    choose_account_from_list,
    choose_account_menu,
)


def render_account_list(console: Console, accounts: list[Account]) -> None:
    """Render the account list or an empty-state message."""
    if not accounts:
        console.print("No accounts found")
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
    prompt_name: Callable[[], str] | None = None,
    prompt_cash: Callable[[], Decimal] | None = None,
) -> None:
    """Add an account via prompts and render confirmation."""
    name_func = prompt_name or (lambda: Prompt.ask("Account name"))
    cash_func = prompt_cash or (lambda: Decimal(Prompt.ask("Cash balance")))
    account = repository.create(name=name_func(), cash_balance=cash_func())
    console.print(f"Added account: {account.name}")


def update_account_flow(
    console: Console,
    repository,
    account: Account,
    prompt_name: Callable[[], str] | None = None,
    prompt_cash: Callable[[], Decimal] | None = None,
) -> None:
    """Update an account via prompts and render confirmation."""
    name_func = prompt_name or (lambda: Prompt.ask("New account name"))
    cash_func = prompt_cash or (lambda: Decimal(Prompt.ask("New cash balance")))
    updated = repository.update(
        account.id,
        name=name_func(),
        cash_balance=cash_func(),
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
) -> None:
    """Run the account management menu loop."""
    selected_account: Account | None = None
    holding_prompt_func = holding_prompt or (lambda: Prompt.ask("Holdings menu"))
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
