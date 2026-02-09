"""Rich-based holdings list rendering."""

from decimal import Decimal, InvalidOperation
from typing import Callable
from uuid import UUID

from rich.console import Console
from rich import box
from rich.panel import Panel
from rich.prompt import Confirm
from rich.table import Table

from portfolio_manager.models import Account, Holding
from portfolio_manager.cli.prompt_select import (
    cancellable_prompt,
    choose_group_from_list,
    choose_holding_from_list,
    choose_holding_menu,
    prompt_decimal,
)


def render_holdings_for_account(
    console: Console,
    repository,
    account: Account,
    stock_lookup: Callable[[UUID], str] | None = None,
) -> None:
    """Render holdings for the given account."""
    holdings: list[Holding] = repository.list_by_account(account.id)
    table = Table(title=f"Holdings in {account.name}", header_style="bold")
    table.add_column("#", style="dim", width=4, justify="right")
    table.add_column("Stock")
    table.add_column("Quantity", justify="right")
    lookup = stock_lookup or (lambda stock_id: str(stock_id))
    for index, holding in enumerate(holdings, start=1):
        table.add_row(str(index), lookup(holding.stock_id), str(holding.quantity))
    console.print(table)


def add_holding_flow(
    console: Console,
    repository,
    account: Account,
    prompt_stock: Callable[[], UUID | str | None] | None = None,
    prompt_quantity: Callable[[], Decimal | None] | None = None,
    stock_repository=None,
    group_repository=None,
    group_chooser: Callable | None = None,
    prompt_group_name: Callable[[], str | None] | None = None,
) -> None:
    """Add a holding via prompts and render confirmation."""
    if prompt_stock is None:

        def stock_func() -> UUID | str | None:
            return cancellable_prompt("Stock ID or Ticker:")
    else:
        stock_func = prompt_stock
    quantity_func = prompt_quantity or (
        lambda: prompt_decimal("Quantity:", console=console)
    )
    stock_value = stock_func()
    if stock_value is None:
        console.print("[yellow]Cancelled[/yellow]")
        return
    try:
        stock_id = (
            stock_value if isinstance(stock_value, UUID) else UUID(str(stock_value))
        )
    except ValueError:
        if stock_repository is None:
            console.print("[red]Unknown stock ticker. Add it first.[/red]")
            return
        stock = stock_repository.get_by_ticker(str(stock_value))
        if stock is None:
            if group_repository is None:
                console.print("[red]Unknown stock ticker. Add it first.[/red]")
                return
            groups = group_repository.list_all()
            if not groups:
                console.print("[yellow]No groups found. Creating one now.[/yellow]")
                name_func = prompt_group_name or (
                    lambda: cancellable_prompt("Group name:")
                )
                group_name = name_func()
                if group_name is None:
                    console.print("[yellow]Cancelled[/yellow]")
                    return
                group = group_repository.create(group_name)
                group_id = group.id
                console.print(
                    Panel.fit(
                        f"[bold]{group.name}[/bold]",
                        title="🧭 Auto-selected Group",
                        border_style="green",
                        box=box.ROUNDED,
                        padding=(0, 2),
                    )
                )
            else:
                group_id = choose_group_from_list(groups, chooser=group_chooser)
                if group_id is None:
                    console.print("[yellow]Cancelled[/yellow]")
                    return
            stock = stock_repository.create(str(stock_value), group_id)
        stock_id = stock.id
    quantity = quantity_func()
    if quantity is None:
        console.print("[yellow]Cancelled[/yellow]")
        return
    holding = repository.create(
        account_id=account.id,
        stock_id=stock_id,
        quantity=quantity,
    )
    console.print(f"Added holding: {holding.quantity}")


def update_holding_flow(
    console: Console,
    repository,
    account: Account,
    holding: Holding,
    prompt_quantity: Callable[[], Decimal | str | None] | None = None,
) -> None:
    """Update a holding quantity via prompt and render confirmation."""
    quantity_func = prompt_quantity or (
        lambda: cancellable_prompt("New quantity:", default=str(holding.quantity))
    )
    quantity_input = quantity_func()
    if quantity_input is None:
        console.print("[yellow]Cancelled[/yellow]")
        return
    if isinstance(quantity_input, Decimal):
        quantity = quantity_input
    else:
        quantity_text = quantity_input.strip()
        if quantity_text == "":
            quantity = holding.quantity
        else:
            try:
                quantity = Decimal(quantity_text)
            except InvalidOperation:
                console.print(
                    "[yellow]Invalid quantity, keeping current value[/yellow]"
                )
                quantity = holding.quantity
    updated = repository.update(holding.id, quantity=quantity)
    console.print(f"Updated holding: {updated.quantity}")


def delete_holding_flow(
    console: Console,
    repository,
    account: Account,
    holding: Holding,
    confirm: Callable[[], bool] | None = None,
) -> None:
    """Delete a holding with confirmation and render status."""
    confirm_func = confirm or (
        lambda: Confirm.ask(
            f"Delete holding {holding.quantity} in {account.name}?", default=False
        )
    )
    if not confirm_func():
        return
    repository.delete(holding.id)
    console.print(f"Deleted holding: {holding.quantity}")


def _select_holding_by_id(holdings: list[Holding], holding_id) -> Holding | None:
    for holding in holdings:
        if holding.id == holding_id:
            return holding
    return None


def run_holdings_menu(
    console: Console,
    repository,
    account: Account,
    prompt: Callable[[], str | None],
    stock_repository=None,
    chooser: Callable | None = None,
    group_repository=None,
    group_chooser: Callable | None = None,
) -> None:
    """Run the holdings menu loop."""
    while True:
        holdings = repository.list_by_account(account.id)
        console.print(
            Panel.fit(
                f"[bold]{account.name}[/bold]",
                title="💼 Current Account",
                border_style="green",
                box=box.ROUNDED,
                padding=(0, 2),
            )
        )
        lookup = None
        if stock_repository is not None:
            group_name_by_id = None
            if group_repository is not None:
                group_name_by_id = {
                    group.id: group.name for group in group_repository.list_all()
                }

            def _lookup(stock_id: UUID) -> str:
                stock = stock_repository.get_by_id(stock_id)
                if stock is None:
                    return str(stock_id)
                if group_name_by_id and stock.group_id in group_name_by_id:
                    return f"{group_name_by_id[stock.group_id]} / {stock.ticker}"
                return stock.ticker

            lookup = _lookup
        render_holdings_for_account(console, repository, account, stock_lookup=lookup)
        action = choose_holding_menu(chooser)
        if action == "back":
            return
        if action == "add":
            add_holding_flow(
                console,
                repository,
                account,
                stock_repository=stock_repository,
                group_repository=group_repository,
                group_chooser=group_chooser,
            )
            continue
        if action == "edit":
            holding_id = choose_holding_from_list(holdings, label_lookup=lookup)
            if holding_id is not None:
                holding = _select_holding_by_id(holdings, holding_id)
                if holding is not None:
                    update_holding_flow(console, repository, account, holding)
            continue
        if action == "delete":
            holding_id = choose_holding_from_list(holdings, label_lookup=lookup)
            if holding_id is not None:
                holding = _select_holding_by_id(holdings, holding_id)
                if holding is not None:
                    delete_holding_flow(console, repository, account, holding)
            continue
