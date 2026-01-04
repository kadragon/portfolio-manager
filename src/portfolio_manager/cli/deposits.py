"""Rich-based deposit management."""

from datetime import datetime, date
from decimal import Decimal

from rich.console import Console
from rich.panel import Panel
from rich.prompt import Confirm, Prompt
from rich.table import Table

from portfolio_manager.models import Account, Deposit
from portfolio_manager.cli.prompt_select import (
    choose_account_from_list,
)


def get_date_input(prompt_text: str = "Date (YYYY-MM-DD)") -> date:
    """Prompt for a date input."""
    while True:
        value = Prompt.ask(prompt_text, default=date.today().isoformat())
        try:
            return datetime.strptime(value, "%Y-%m-%d").date()
        except ValueError:
            print("Invalid format. Please use YYYY-MM-DD.")


def render_deposit_list(
    console: Console, deposits: list[Deposit], account: Account
) -> None:
    """Render the deposit list."""
    if not deposits:
        console.print(f"No deposits found for {account.name}")
        # Show table header anyway or just return?
        # Better to show table header so user knows context.

    table = Table(title=f"Deposits for {account.name}", header_style="bold")
    table.add_column("#", style="dim", width=4, justify="right")
    table.add_column("Date", justify="center")
    table.add_column("Amount", justify="right")
    table.add_column("Note")

    total = Decimal("0")
    for index, deposit in enumerate(deposits, start=1):
        table.add_row(
            str(index),
            deposit.deposit_date.isoformat(),
            f"{deposit.amount:,.0f}",
            deposit.note or "",
        )
        total += deposit.amount

    if deposits:
        table.add_section()
        table.add_row("", "Total", f"{total:,.0f}", "", style="bold")

    console.print(table)


def add_deposit_flow(
    console: Console,
    deposit_repository,
    account: Account,
) -> None:
    """Add a deposit."""
    while True:
        try:
            amount_str = Prompt.ask("Amount")
            amount = Decimal(amount_str)
            break
        except Exception:
            console.print("[red]Invalid amount[/red]")

    deposit_date = get_date_input()
    note = Prompt.ask("Note", default="")

    deposit_repository.create(
        account_id=account.id,
        amount=amount,
        deposit_date=deposit_date,
        note=note if note else None,
    )
    console.print(f"[green]Added deposit of {amount:,.0f} to {account.name}[/green]")


def delete_deposit_flow(
    console: Console,
    deposit_repository,
    deposits: list[Deposit],
) -> None:
    """Delete a deposit."""
    if not deposits:
        console.print("[yellow]No deposits to delete[/yellow]")
        return

    choice = Prompt.ask(
        "Select deposit # to delete (or 'c' to cancel)",
        choices=[str(i) for i in range(1, len(deposits) + 1)] + ["c"],
    )

    if choice == "c":
        return

    index = int(choice) - 1
    deposit = deposits[index]

    if Confirm.ask(
        f"Delete deposit of {deposit.amount:,.0f} from {deposit.deposit_date}?",
        default=False,
    ):
        deposit_repository.delete(deposit.id)
        console.print("[green]Deleted deposit[/green]")


def run_deposit_menu(
    console: Console,
    deposit_repository,
    account_repository,
) -> None:
    """Run the deposit management menu."""
    while True:
        accounts = account_repository.list_all()
        if not accounts:
            console.print("No accounts found.")
            return

        console.print(Panel("Select an account to manage deposits", title="Deposits"))
        account_id = choose_account_from_list(accounts)

        if account_id is None:  # Back/Exit logic in prompt_select might return None?
            # choose_account_from_list returns None if user selects 'c' or similar if implemented
            # Let's check choose_account_from_list implementation.
            return

        account = next((a for a in accounts if a.id == account_id), None)
        if not account:
            continue

        while True:
            deposits = deposit_repository.list_by_account(account.id)
            render_deposit_list(console, deposits, account)

            action = Prompt.ask(
                "Action", choices=["add", "delete", "back"], default="back"
            )

            if action == "back":
                break
            elif action == "add":
                add_deposit_flow(console, deposit_repository, account)
            elif action == "delete":
                delete_deposit_flow(console, deposit_repository, deposits)
