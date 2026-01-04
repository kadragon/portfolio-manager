"""Rich-only CLI entrypoint."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Iterable

from dotenv import load_dotenv
from rich.console import Console
from rich import box
from rich.panel import Panel
from rich.prompt import Prompt

from portfolio_manager.cli.app import render_dashboard, render_main_menu
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

    try:
        while True:
            render_main_menu(console)

            # Render dashboard
            portfolio_service = container.get_portfolio_service()

            # Show dashboard with or without prices
            if container.price_service:
                try:
                    summary = portfolio_service.get_portfolio_summary()
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
                continue
            if action == "quit":
                return
    finally:
        container.close()
