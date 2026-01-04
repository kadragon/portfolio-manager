"""Rich-based deposit management."""

from datetime import datetime, date
from decimal import Decimal

from rich.console import Console
from rich.prompt import Confirm, Prompt
from rich.table import Table

from portfolio_manager.models import Deposit


def get_date_input(
    prompt_text: str = "Date (YYYY-MM-DD)", default: date | None = None
) -> date:
    """Prompt for a date input."""
    default_str = default.isoformat() if default else date.today().isoformat()
    while True:
        value = Prompt.ask(prompt_text, default=default_str)
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
) -> None:
    """Add a deposit."""
    deposit_date = get_date_input()

    existing = deposit_repository.get_by_date(deposit_date)
    if existing:
        console.print(f"[red]Deposit for {deposit_date} already exists.[/red]")
        if Confirm.ask("Do you want to modify it instead?", default=True):
            update_deposit_flow(console, deposit_repository, existing)
        return

    while True:
        try:
            amount_str = Prompt.ask("Amount")
            amount = Decimal(amount_str)
            break
        except Exception:
            console.print("[red]Invalid amount[/red]")

    note = Prompt.ask("Note", default="")

    deposit_repository.create(
        amount=amount, deposit_date=deposit_date, note=note if note else None
    )
    console.print(f"[green]Added deposit for {deposit_date}[/green]")


def update_deposit_flow(
    console: Console,
    deposit_repository,
    deposit: Deposit,
) -> None:
    """Update a deposit."""
    console.print(f"Updating deposit for {deposit.deposit_date}")

    amount_str = Prompt.ask("New Amount", default=str(deposit.amount))
    try:
        amount = Decimal(amount_str)
    except Exception:
        console.print("[red]Invalid amount, keeping original[/red]")
        amount = deposit.amount

    note = Prompt.ask("New Note", default=deposit.note or "")

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

        action = Prompt.ask(
            "Action", choices=["add", "edit", "delete", "back"], default="back"
        )

        if action == "back":
            break
        elif action == "add":
            add_deposit_flow(console, deposit_repository)
        elif action == "edit":
            select_deposit_to_edit(console, deposit_repository, deposits)
        elif action == "delete":
            delete_deposit_flow(console, deposit_repository, deposits)
