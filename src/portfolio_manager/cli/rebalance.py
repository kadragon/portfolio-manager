"""Rebalance CLI rendering."""

from rich.console import Console
from rich.panel import Panel
from rich.table import Table

from portfolio_manager.models.rebalance import GroupRebalanceSignal


def render_rebalance_actions(
    console: Console, actions: list[GroupRebalanceSignal]
) -> None:
    """Render group-level rebalance action table."""
    if not actions:
        console.print(
            Panel(
                "Portfolio is balanced. No rebalancing needed.",
                title="Rebalance",
                border_style="green",
            )
        )
        return

    table = Table(title="Rebalance Actions")
    table.add_column("Group", style="blue")
    table.add_column("Action", style="white")
    table.add_column("Delta", style="magenta", justify="right")
    table.add_column("Manual Review", style="yellow")

    for signal in actions:
        action_label = signal.action.name.replace("_", " ")
        delta_label = f"{signal.delta:.2f}%"
        manual_label = "Yes" if signal.manual_review_required else "No"
        table.add_row(signal.group.name, action_label, delta_label, manual_label)

    console.print(table)
