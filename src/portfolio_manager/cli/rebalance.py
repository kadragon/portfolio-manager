"""Rebalance recommendations CLI rendering."""

from decimal import Decimal, ROUND_HALF_UP

from rich.console import Console
from rich.panel import Panel
from rich.table import Table

from portfolio_manager.models.rebalance import RebalanceRecommendation


def render_rebalance_recommendations(
    console: Console,
    sell_recommendations: list[RebalanceRecommendation],
    buy_recommendations: list[RebalanceRecommendation],
) -> None:
    """Render rebalance recommendations tables."""

    def format_quantity(quantity: Decimal | None) -> str:
        if quantity is None:
            return "-"
        rounded = quantity.quantize(Decimal("1"), rounding=ROUND_HALF_UP)
        return f"{rounded:,}"

    def format_stock_name(stock_name: str | None, ticker: str) -> str:
        name = stock_name or ticker
        return name.replace("증권상장지수투자신탁(주식)", "").strip()

    account_label = "Same Account"

    if not sell_recommendations and not buy_recommendations:
        console.print(
            Panel(
                "Portfolio is balanced. No rebalancing needed.",
                title="Rebalance",
                border_style="green",
            )
        )
        return

    if sell_recommendations:
        sell_table = Table(title="SELL Recommendations (Overseas First)")
        sell_table.add_column("Ticker", style="red")
        sell_table.add_column("Name", style="white")
        sell_table.add_column("Group", style="blue")
        sell_table.add_column("Sell Account", style="cyan")
        sell_table.add_column("Quantity", style="magenta", justify="right")
        sell_table.add_column("Amount", style="yellow", justify="right")

        for rec in sell_recommendations:
            currency_symbol = "$" if rec.currency == "USD" else "₩"
            sell_table.add_row(
                rec.ticker,
                format_stock_name(rec.stock_name, rec.ticker),
                rec.group_name or "-",
                account_label,
                format_quantity(rec.quantity),
                f"{currency_symbol}{rec.amount:,.0f}",
            )

        console.print(sell_table)

    if buy_recommendations:
        buy_table = Table(title="BUY Recommendations (Domestic First)")
        buy_table.add_column("Ticker", style="green")
        buy_table.add_column("Name", style="white")
        buy_table.add_column("Group", style="blue")
        buy_table.add_column("Buy Account", style="cyan")
        buy_table.add_column("Quantity", style="magenta", justify="right")
        buy_table.add_column("Amount", style="yellow", justify="right")

        for rec in buy_recommendations:
            currency_symbol = "$" if rec.currency == "USD" else "₩"
            buy_table.add_row(
                rec.ticker,
                format_stock_name(rec.stock_name, rec.ticker),
                rec.group_name or "-",
                account_label,
                format_quantity(rec.quantity),
                f"{currency_symbol}{rec.amount:,.0f}",
            )

        console.print(buy_table)
