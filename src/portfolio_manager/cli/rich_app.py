"""Rich-based CLI rendering."""

from rich.console import Console
from rich.table import Table

from portfolio_manager.services.portfolio_service import GroupHoldings


def render_main_menu(console: Console) -> None:
    """Render the main menu."""
    console.print("Portfolio Manager")


def render_dashboard(console: Console, group_holdings: list[GroupHoldings]) -> None:
    """Render dashboard with groups and their stock holdings."""
    for group_holding in group_holdings:
        if not group_holding.stock_holdings:
            continue

        table = Table(title=f"📊 {group_holding.group.name}")
        table.add_column("Ticker", style="cyan")
        table.add_column("Quantity", style="magenta", justify="right")

        for stock_holding in group_holding.stock_holdings:
            table.add_row(
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
