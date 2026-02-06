"""Rich-based stock list rendering."""

from typing import Callable

from rich.console import Console
from rich import box
from rich.panel import Panel
from rich.prompt import Confirm
from rich.table import Table

from portfolio_manager.models import Group, Stock
from portfolio_manager.cli.prompt_select import (
    cancellable_prompt,
    choose_group_from_list,
    choose_stock_from_list,
    choose_stock_menu,
)


def render_stocks_for_group(console: Console, repository, group: Group) -> None:
    """Render stocks for the given group."""
    stocks: list[Stock] = repository.list_by_group(group.id)
    table = Table(title=f"Stocks in {group.name}", header_style="bold")
    table.add_column("#", style="dim", width=4, justify="right")
    table.add_column("Ticker", style="bold")
    for index, stock in enumerate(stocks, start=1):
        table.add_row(str(index), stock.ticker)
    console.print(table)


def add_stock_flow(
    console: Console,
    repository,
    group: Group,
    prompt: Callable[[], str | None] | None = None,
) -> None:
    """Add a stock via prompt and render confirmation."""
    prompt_func = prompt or (lambda: cancellable_prompt("Stock ticker:"))
    ticker = prompt_func()
    if ticker is None:
        console.print("[yellow]Cancelled[/yellow]")
        return
    stock = repository.create(ticker, group.id)
    console.print(f"Added stock: {stock.ticker}")


def update_stock_flow(
    console: Console,
    repository,
    stock: Stock,
    prompt: Callable[[], str | None] | None = None,
) -> None:
    """Update a stock ticker via prompt and render confirmation."""
    prompt_func = prompt or (
        lambda: cancellable_prompt("New stock ticker:", default=stock.ticker)
    )
    ticker_input = prompt_func()
    if ticker_input is None:
        console.print("[yellow]Cancelled[/yellow]")
        return
    ticker = stock.ticker if ticker_input.strip() == "" else ticker_input
    updated = repository.update(stock.id, ticker)
    console.print(f"Updated stock: {updated.ticker}")


def move_stock_flow(
    console: Console,
    repository,
    stock: Stock,
    group: Group,
) -> None:
    """Move a stock to a new group and render confirmation."""
    updated = repository.update_group(stock.id, group.id)
    console.print(f"Moved stock: {updated.ticker} → {group.name}")


def delete_stock_flow(
    console: Console,
    repository,
    group: Group,
    stock: Stock,
    confirm: Callable[[], bool] | None = None,
) -> None:
    """Delete a stock with confirmation and render status."""
    confirm_func = confirm or (
        lambda: Confirm.ask(f"Delete {stock.ticker}?", default=False)
    )
    if not confirm_func():
        return
    repository.delete(stock.id)
    console.print(f"Deleted stock: {stock.ticker}")


def _select_stock_by_id(stocks: list[Stock], stock_id) -> Stock | None:
    for stock in stocks:
        if stock.id == stock_id:
            return stock
    return None


def run_stock_menu(
    console: Console,
    repository,
    group: Group,
    prompt: Callable[[], str],
    chooser: Callable[[], str] | None = None,
    group_repository=None,
    group_chooser: Callable | None = None,
) -> None:
    """Run the stock menu loop."""
    while True:
        console.print(
            Panel.fit(
                f"[bold]{group.name}[/bold]",
                title="🗂 Current Group",
                border_style="cyan",
                box=box.ROUNDED,
                padding=(0, 2),
            )
        )
        render_stocks_for_group(console, repository, group)
        action = choose_stock_menu(chooser)
        if action == "back":
            return
        if action == "add":
            add_stock_flow(console, repository, group)
            continue
        if action == "edit":
            stock_id = choose_stock_from_list(repository.list_by_group(group.id))
            if stock_id is not None:
                stock = _select_stock_by_id(
                    repository.list_by_group(group.id), stock_id
                )
                if stock is not None:
                    update_stock_flow(console, repository, stock)
            continue
        if action == "move":
            if group_repository is None:
                console.print("[red]No groups available for move.[/red]")
                continue
            stock_id = choose_stock_from_list(repository.list_by_group(group.id))
            if stock_id is not None:
                stock = _select_stock_by_id(
                    repository.list_by_group(group.id), stock_id
                )
                if stock is None:
                    continue
                groups = group_repository.list_all()
                target_group_id = choose_group_from_list(groups, chooser=group_chooser)
                if target_group_id is None:
                    continue
                target_group = next(
                    (item for item in groups if item.id == target_group_id), None
                )
                if target_group is not None:
                    move_stock_flow(console, repository, stock, target_group)
            continue
        if action == "delete":
            stock_id = choose_stock_from_list(repository.list_by_group(group.id))
            if stock_id is not None:
                stock = _select_stock_by_id(
                    repository.list_by_group(group.id), stock_id
                )
                if stock is not None:
                    delete_stock_flow(console, repository, group, stock)
            continue
