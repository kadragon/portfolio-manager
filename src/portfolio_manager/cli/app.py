"""Rich-based CLI rendering."""

from decimal import Decimal, ROUND_HALF_UP

from rich.console import Console
from rich.table import Table

from portfolio_manager.models import Group
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

    def truncate_name(name: str, max_length: int = 25) -> str:
        if len(name) <= max_length:
            return name
        return name[:max_length]

    # Handle PortfolioSummary
    if isinstance(data, PortfolioSummary):
        if not data.holdings:
            console.print("No groups found. Create a group to get started.")
            return

        def format_quantity(quantity: Decimal, currency: str) -> str:
            if currency == "KRW":
                return str(quantity)
            return str(quantity.quantize(Decimal("1"), rounding=ROUND_HALF_UP))

        def format_percent(value: Decimal) -> str:
            return str(value.quantize(Decimal("0.1"), rounding=ROUND_HALF_UP))

        def format_signed_percent(value: Decimal) -> str:
            sign = "+" if value > 0 else "-" if value < 0 else ""
            return f"{sign}{format_percent(abs(value))}"

        table = Table(title="📊 Portfolio")
        table.add_column("Group", style="blue")
        table.add_column("Ticker", style="cyan")
        table.add_column("Name", style="white")
        table.add_column("Quantity", style="magenta", justify="right")
        table.add_column("Price", style="green", justify="right")
        table.add_column("Value", style="yellow", justify="right")

        group_totals: dict[str, Decimal] = {}
        group_lookup: dict[str, Group] = {}
        for group, holding_with_price in data.holdings:
            currency_symbol = "₩" if holding_with_price.currency == "KRW" else "$"
            display_name = (
                holding_with_price.name
                if holding_with_price.name
                else holding_with_price.stock.ticker
            )
            value_krw = holding_with_price.value_krw
            if value_krw is None:
                value_krw = holding_with_price.value
            group_key = str(group.id)
            if group_key not in group_totals:
                group_totals[group_key] = Decimal("0")
                group_lookup[group_key] = group
            group_totals[group_key] = group_totals[group_key] + value_krw
            table.add_row(
                group.name,
                holding_with_price.stock.ticker,
                truncate_name(display_name),
                format_quantity(
                    holding_with_price.quantity, holding_with_price.currency
                ),
                f"{currency_symbol}{holding_with_price.price:,.0f}"
                if holding_with_price.currency == "KRW"
                else f"{currency_symbol}{holding_with_price.price:,.2f}",
                f"₩{value_krw:,.0f}",
            )

        console.print(table)

        summary_table = Table(title="Group Summary")
        summary_table.add_column("Group", style="blue", no_wrap=True)
        summary_table.add_column("Total", style="yellow", justify="right", no_wrap=True)
        summary_table.add_column(
            "% of Total", style="green", justify="right", no_wrap=True
        )
        summary_table.add_column(
            "Target %", style="cyan", justify="right", no_wrap=True
        )
        summary_table.add_column(
            "Diff %", style="magenta", justify="right", no_wrap=True
        )
        summary_table.add_column("Action", style="white", justify="left", no_wrap=True)
        summary_table.add_column("Amount", style="white", justify="right", no_wrap=True)

        total_value = data.total_value
        for group_key, group_total in group_totals.items():
            group = group_lookup[group_key]
            if total_value > 0:
                actual_percent = (group_total / total_value) * Decimal("100")
            else:
                actual_percent = Decimal("0")
            target_percent = Decimal(str(group.target_percentage))
            diff_percent = actual_percent - target_percent
            target_value = (total_value * target_percent) / Decimal("100")
            diff_value = group_total - target_value
            if diff_value > 0:
                action = "[red]🔴 Sell[/red]"
                amount = f"[red]₩{diff_value:,.0f}[/red]"
                diff_label = f"[red]{format_signed_percent(diff_percent)}%[/red]"
            elif diff_value < 0:
                action = "[green]🟢 Buy[/green]"
                amount = f"[green]₩{abs(diff_value):,.0f}[/green]"
                diff_label = f"[green]{format_signed_percent(diff_percent)}%[/green]"
            else:
                action = "-"
                amount = "-"
                diff_label = f"{format_signed_percent(diff_percent)}%"
            summary_table.add_row(
                group.name,
                f"₩{group_total:,.0f}",
                f"{format_percent(actual_percent)}%",
                f"{format_percent(target_percent)}%",
                diff_label,
                action,
                amount,
            )

        console.print(summary_table)
        console.print(f"\n[bold]Total Value: ₩{data.total_value:,.0f}[/bold]")
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
