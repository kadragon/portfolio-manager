"""Rebalance recommendations CLI rendering."""

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
        sell_table.add_column("Priority", style="dim", justify="center")
        sell_table.add_column("Ticker", style="red")
        sell_table.add_column("Group", style="blue")
        sell_table.add_column("Amount", style="yellow", justify="right")

        for rec in sell_recommendations:
            currency_symbol = "$" if rec.currency == "USD" else "₩"
            sell_table.add_row(
                str(rec.priority),
                rec.ticker,
                rec.group_name or "-",
                f"{currency_symbol}{rec.amount:,.0f}",
            )

        console.print(sell_table)

    if buy_recommendations:
        buy_table = Table(title="BUY Recommendations (Domestic First)")
        buy_table.add_column("Priority", style="dim", justify="center")
        buy_table.add_column("Ticker", style="green")
        buy_table.add_column("Group", style="blue")
        buy_table.add_column("Amount", style="yellow", justify="right")

        for rec in buy_recommendations:
            currency_symbol = "$" if rec.currency == "USD" else "₩"
            buy_table.add_row(
                str(rec.priority),
                rec.ticker,
                rec.group_name or "-",
                f"{currency_symbol}{rec.amount:,.0f}",
            )

        console.print(buy_table)
