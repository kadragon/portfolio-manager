"""Rich-based deposit management."""

from datetime import datetime, date
from decimal import Decimal
from typing import Callable

from rich.console import Console
from rich.prompt import Confirm, Prompt
from rich.table import Table

from portfolio_manager.models import Deposit
from portfolio_manager.cli.prompt_select import cancellable_prompt, choose_deposit_menu


def get_date_input(
    prompt_text: str = "Date (YYYY-MM-DD)",
    default: date | None = None,
    prompt_func: Callable[[], str | None] | None = None,
) -> date | None:
    """Prompt for a date input. Returns None if cancelled."""
    default_str = default.isoformat() if default else date.today().isoformat()

    def default_prompt() -> str | None:
        return cancellable_prompt(f"{prompt_text}:", default=default_str)

    actual_func = prompt_func if prompt_func is not None else default_prompt

    while True:
        value = actual_func()
        if value is None:
            return None
        try:
            return datetime.strptime(value, "%Y-%m-%d").date()
        except ValueError:
            print("Invalid format. Please use YYYY-MM-DD.")


def render_deposit_list(console: Console, deposits: list[Deposit]) -> None:
    """Render the deposit list."""
    if not deposits:
        console.print("No deposits found")

    table = Table(title="Deposits", header_style="bold")
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
    prompt_date=None,
    prompt_amount=None,
    prompt_note=None,
) -> None:
    """Add a deposit."""
    deposit_date = get_date_input(prompt_func=prompt_date)
    if deposit_date is None:
        console.print("[yellow]Cancelled[/yellow]")
        return

    existing = deposit_repository.get_by_date(deposit_date)
    if existing:
        console.print(f"[red]Deposit for {deposit_date} already exists.[/red]")
        if Confirm.ask("Do you want to modify it instead?", default=True):
            update_deposit_flow(console, deposit_repository, existing)
        return

    amount_func = prompt_amount or (lambda: cancellable_prompt("Amount:"))
    while True:
        try:
            amount_str = amount_func()
            if amount_str is None:
                console.print("[yellow]Cancelled[/yellow]")
                return
            amount = Decimal(amount_str)
            break
        except Exception:
            console.print("[red]Invalid amount[/red]")

    note_func = prompt_note or (lambda: cancellable_prompt("Note:", default=""))
    note = note_func()
    if note is None:
        console.print("[yellow]Cancelled[/yellow]")
        return

    deposit_repository.create(
        amount=amount, deposit_date=deposit_date, note=note if note else None
    )
    console.print(f"[green]Added deposit for {deposit_date}[/green]")


def update_deposit_flow(
    console: Console,
    deposit_repository,
    deposit: Deposit,
    prompt_amount=None,
    prompt_note=None,
) -> None:
    """Update a deposit."""
    console.print(f"Updating deposit for {deposit.deposit_date}")

    amount_func = prompt_amount or (
        lambda: cancellable_prompt("New Amount:", default=str(deposit.amount))
    )
    amount_str = amount_func()
    if amount_str is None:
        console.print("[yellow]Cancelled[/yellow]")
        return
    try:
        amount = Decimal(amount_str)
    except Exception:
        console.print("[red]Invalid amount, keeping original[/red]")
        amount = deposit.amount

    note_func = prompt_note or (
        lambda: cancellable_prompt("New Note:", default=deposit.note or "")
    )
    note = note_func()
    if note is None:
        console.print("[yellow]Cancelled[/yellow]")
        return

    deposit_repository.update(deposit.id, amount=amount, note=note if note else None)
    console.print("[green]Updated deposit[/green]")


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


def select_deposit_to_edit(
    console: Console,
    deposit_repository,
    deposits: list[Deposit],
) -> None:
    """Select a deposit to edit."""
    if not deposits:
        console.print("[yellow]No deposits to edit[/yellow]")
        return

    choice = Prompt.ask(
        "Select deposit # to edit (or 'c' to cancel)",
        choices=[str(i) for i in range(1, len(deposits) + 1)] + ["c"],
    )

    if choice == "c":
        return

    index = int(choice) - 1
    deposit = deposits[index]
    update_deposit_flow(console, deposit_repository, deposit)


def run_deposit_menu(
    console: Console,
    deposit_repository,
) -> None:
    """Run the deposit management menu."""
    while True:
        deposits = deposit_repository.list_all()
        render_deposit_list(console, deposits)

        action = choose_deposit_menu()

        if action == "back" or action is None:
            break
        elif action == "add":
            add_deposit_flow(console, deposit_repository)
        elif action == "edit":
            select_deposit_to_edit(console, deposit_repository, deposits)
        elif action == "delete":
            delete_deposit_flow(console, deposit_repository, deposits)
