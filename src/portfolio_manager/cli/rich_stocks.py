"""Rich-based stock list rendering."""

from typing import Callable

from rich.console import Console
from rich.table import Table
from rich.prompt import Confirm, Prompt

from portfolio_manager.models import Group, Stock


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
    prompt: Callable[[], str] | None = None,
) -> None:
    """Add a stock via prompt and render confirmation."""
    prompt_func = prompt or (lambda: Prompt.ask("Stock ticker"))
    ticker = prompt_func()
    stock = repository.create(ticker, group.id)
    console.print(f"Added stock: {stock.ticker}")


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


def run_stock_menu(
    console: Console,
    repository,
    group: Group,
    prompt: Callable[[], str],
) -> None:
    """Run the stock menu loop."""
    while True:
        render_stocks_for_group(console, repository, group)
        console.print("[bold]Options:[/bold] a=add, d=delete, b=back")
        choice = prompt().strip().lower()
        if choice == "b":
            return
