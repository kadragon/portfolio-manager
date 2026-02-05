"""Rich-only CLI entrypoint."""

from __future__ import annotations

from dataclasses import dataclass
import time
from typing import Iterable

from dotenv import load_dotenv
from rich.console import Console
from rich import box
from rich.panel import Panel
from rich.prompt import Prompt

from portfolio_manager.cli.app import render_dashboard, render_main_menu
from portfolio_manager.cli.deposits import run_deposit_menu
from portfolio_manager.cli.rebalance import render_rebalance_recommendations
from portfolio_manager.cli.groups import (
    add_group_flow,
    delete_group_flow,
    render_group_list,
    update_group_flow,
)
from portfolio_manager.cli.prompt_select import (
    choose_group_from_list,
    choose_group_menu,
    choose_main_menu,
)
from portfolio_manager.cli.accounts import run_account_menu
from portfolio_manager.cli.stocks import run_stock_menu
from portfolio_manager.core.container import ServiceContainer
from portfolio_manager.models import Group
from portfolio_manager.services.portfolio_service import PortfolioSummary
from portfolio_manager.services.rebalance_service import RebalanceService

_SUMMARY_CACHE_TTL_SECONDS = 30.0


@dataclass(frozen=True)
class GroupMenuState:
    """State for group menu."""

    groups: list[Group]


def _load_groups(container: ServiceContainer) -> list[Group]:
    return container.group_repository.list_all()


def _select_group_by_index(groups: Iterable[Group], index: int) -> Group | None:
    if index < 1:
        return None
    group_list = list(groups)
    if index > len(group_list):
        return None
    return group_list[index - 1]


def _select_group_by_id(groups: Iterable[Group], group_id) -> Group | None:
    for group in groups:
        if group.id == group_id:
            return group
    return None


def run_rebalance_menu(console: Console, container: ServiceContainer) -> None:
    """Run the rebalance recommendations view."""
    portfolio_service = container.get_portfolio_service()

    if not container.price_service:
        console.print(
            "[yellow]Price service not available. Cannot calculate rebalance.[/yellow]"
        )
        return

    try:
        summary = portfolio_service.get_portfolio_summary()
    except Exception as e:
        console.print(f"[red]Error fetching portfolio: {e}[/red]")
        return

    rebalance_service = RebalanceService()
    sell_recommendations = rebalance_service.get_sell_recommendations(summary)
    buy_recommendations = rebalance_service.get_buy_recommendations(summary)

    render_rebalance_recommendations(console, sell_recommendations, buy_recommendations)

    Prompt.ask("Press Enter to continue")


def run_group_menu(console: Console, container: ServiceContainer) -> None:
    """Run the group management menu loop."""
    selected_group: Group | None = None
    while True:
        groups = _load_groups(container)
        render_group_list(console, groups)
        if selected_group is not None:
            console.print(
                Panel.fit(
                    f"[bold]{selected_group.name}[/bold]",
                    title="🗂 Current Group",
                    border_style="cyan",
                    box=box.ROUNDED,
                    padding=(0, 2),
                )
            )
        action = choose_group_menu()
        if action == "back":
            return
        if action == "add":
            add_group_flow(console, container.group_repository)
            continue
        if action == "edit":
            group_id = choose_group_from_list(groups)
            if group_id is not None:
                group = _select_group_by_id(groups, group_id)
                if group is not None:
                    update_group_flow(console, container.group_repository, group)
            continue
        if action == "delete":
            group_id = choose_group_from_list(groups)
            if group_id is not None:
                group = _select_group_by_id(groups, group_id)
                if group is not None:
                    delete_group_flow(console, container.group_repository, group)
            continue
        if action == "select":
            group_id = choose_group_from_list(groups)
            if group_id is not None:
                group = _select_group_by_id(groups, group_id)
                if group is not None:
                    selected_group = group
                    run_stock_menu(
                        console,
                        container.stock_repository,
                        group,
                        prompt=lambda: Prompt.ask("Stock menu"),
                        group_repository=container.group_repository,
                    )
            continue


def main() -> None:
    """CLI entrypoint."""
    load_dotenv()
    console = Console()
    container = ServiceContainer(console)
    container.setup()

    summary_cache: PortfolioSummary | None = None
    summary_cached_at: float | None = None

    def invalidate_summary_cache() -> None:
        nonlocal summary_cache, summary_cached_at
        summary_cache = None
        summary_cached_at = None

    try:
        while True:
            render_main_menu(console)

            # Render dashboard
            portfolio_service = container.get_portfolio_service()

            # Show dashboard with or without prices
            if container.price_service:
                try:
                    now = time.time()
                    if (
                        summary_cache is not None
                        and summary_cached_at is not None
                        and now - summary_cached_at < _SUMMARY_CACHE_TTL_SECONDS
                    ):
                        summary = summary_cache
                    else:
                        summary = portfolio_service.get_portfolio_summary(
                            include_change_rates=False
                        )
                        summary_cache = summary
                        summary_cached_at = now
                    render_dashboard(console, summary)
                except Exception as e:
                    console.print(
                        f"[yellow]Warning: Could not fetch prices: {e}[/yellow]"
                    )
                    group_holdings = portfolio_service.get_holdings_by_group()
                    render_dashboard(console, group_holdings)
            else:
                group_holdings = portfolio_service.get_holdings_by_group()
                render_dashboard(console, group_holdings)

            action = choose_main_menu()
            if action == "groups":
                run_group_menu(console, container)
                invalidate_summary_cache()
                continue
            if action == "accounts":
                run_account_menu(
                    console,
                    container.account_repository,
                    container.holding_repository,
                    prompt=lambda: Prompt.ask("Accounts menu"),
                    stock_repository=container.stock_repository,
                    group_repository=container.group_repository,
                )
                invalidate_summary_cache()
                continue
            if action == "deposits":
                run_deposit_menu(
                    console,
                    container.deposit_repository,
                )
                invalidate_summary_cache()
                continue
            if action == "rebalance":
                run_rebalance_menu(console, container)
                invalidate_summary_cache()
                continue
            if action == "quit":
                return
    finally:
        container.close()
