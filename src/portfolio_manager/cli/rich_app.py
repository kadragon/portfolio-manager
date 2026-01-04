"""Rich-based CLI rendering."""

from rich.console import Console
from rich.table import Table

from portfolio_manager.services.portfolio_service import (
    GroupHoldings,
    PortfolioSummary,
)


def render_main_menu(console: Console) -> None:
    """Render the main menu."""
    console.print("Portfolio Manager")


def render_dashboard(
    console: Console, data: list[GroupHoldings] | PortfolioSummary
) -> None:
    """Render dashboard with groups and their stock holdings."""
    # Handle PortfolioSummary
    if isinstance(data, PortfolioSummary):
        if not data.holdings:
            console.print("No groups found. Create a group to get started.")
            return

        table = Table(title="📊 Portfolio")
        table.add_column("Group", style="blue")
        table.add_column("Ticker", style="cyan")
        table.add_column("Quantity", style="magenta", justify="right")
        table.add_column("Price", style="green", justify="right")
        table.add_column("Value", style="yellow", justify="right")

        for group, holding_with_price in data.holdings:
            table.add_row(
                group.name,
                holding_with_price.stock.ticker,
                str(holding_with_price.quantity),
                f"${holding_with_price.price}",
                f"${holding_with_price.value:,.2f}",
            )

        console.print(table)
        console.print(f"\n[bold]Total Value: ${data.total_value:,.2f}[/bold]")
        return

    # Handle list[GroupHoldings]
    group_holdings = data
    if not group_holdings:
        console.print("No groups found. Create a group to get started.")
        return

    # Single table for all stocks
    table = Table(title="📊 Portfolio")
    table.add_column("Group", style="blue")
    table.add_column("Ticker", style="cyan")
    table.add_column("Quantity", style="magenta", justify="right")

    for group_holding in group_holdings:
        if not group_holding.stock_holdings:
            table.add_row(group_holding.group.name, "(no stocks)", "-")
        else:
            for stock_holding in group_holding.stock_holdings:
                table.add_row(
                    group_holding.group.name,
                    stock_holding.stock.ticker,
                    str(stock_holding.quantity),
                )

    console.print(table)


def select_main_menu_option(choice: str) -> str | None:
    """Map a menu choice to an action."""
    normalized = choice.strip().lower()
    if normalized == "g":
        return "groups"
    if normalized == "a":
        return "accounts"
    return None


def handle_main_menu_key(key: str) -> str | None:
    """Handle a main menu key press."""
    return select_main_menu_option(key)
