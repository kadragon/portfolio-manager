"""Rich-only CLI entrypoint."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Iterable

from dotenv import load_dotenv
from rich.console import Console
from rich import box
from rich.panel import Panel
from rich.prompt import Prompt

from portfolio_manager.cli.rich_app import render_dashboard, render_main_menu
from portfolio_manager.cli.rich_groups import (
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
from portfolio_manager.cli.rich_accounts import run_account_menu
from portfolio_manager.cli.rich_stocks import run_stock_menu
from portfolio_manager.models import Group
from portfolio_manager.repositories.account_repository import AccountRepository
from portfolio_manager.repositories.group_repository import GroupRepository
from portfolio_manager.repositories.holding_repository import HoldingRepository
from portfolio_manager.repositories.stock_repository import StockRepository
from portfolio_manager.services.portfolio_service import PortfolioService
from portfolio_manager.services.supabase_client import get_supabase_client


@dataclass(frozen=True)
class GroupMenuState:
    """State for group menu."""

    groups: list[Group]


def _load_groups() -> list[Group]:
    client = get_supabase_client()
    repo = GroupRepository(client)
    return repo.list_all()


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


def run_group_menu(console: Console) -> None:
    """Run the group management menu loop."""
    selected_group: Group | None = None
    while True:
        groups = _load_groups()
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
            client = get_supabase_client()
            repo = GroupRepository(client)
            add_group_flow(console, repo)
            continue
        if action == "edit":
            group_id = choose_group_from_list(groups)
            if group_id is not None:
                group = _select_group_by_id(groups, group_id)
                if group is not None:
                    client = get_supabase_client()
                    repo = GroupRepository(client)
                    update_group_flow(console, repo, group)
            continue
        if action == "delete":
            group_id = choose_group_from_list(groups)
            if group_id is not None:
                group = _select_group_by_id(groups, group_id)
                if group is not None:
                    client = get_supabase_client()
                    repo = GroupRepository(client)
                    delete_group_flow(console, repo, group)
            continue
        if action == "select":
            group_id = choose_group_from_list(groups)
            if group_id is not None:
                group = _select_group_by_id(groups, group_id)
                if group is not None:
                    selected_group = group
                    client = get_supabase_client()
                    repo = StockRepository(client)
                    group_repo = GroupRepository(client)
                    run_stock_menu(
                        console,
                        repo,
                        group,
                        prompt=lambda: Prompt.ask("Stock menu"),
                        group_repository=group_repo,
                    )
            continue


def main() -> None:
    """CLI entrypoint."""
    load_dotenv()
    console = Console()
    while True:
        render_main_menu(console)

        # Render dashboard
        client = get_supabase_client()
        group_repo = GroupRepository(client)
        stock_repo = StockRepository(client)
        holding_repo = HoldingRepository(client)
        portfolio_service = PortfolioService(group_repo, stock_repo, holding_repo)
        group_holdings = portfolio_service.get_holdings_by_group()
        render_dashboard(console, group_holdings)

        action = choose_main_menu()
        if action == "groups":
            run_group_menu(console)
            continue
        if action == "accounts":
            account_repo = AccountRepository(client)
            run_account_menu(
                console,
                account_repo,
                holding_repo,
                prompt=lambda: Prompt.ask("Accounts menu"),
                stock_repository=stock_repo,
                group_repository=group_repo,
            )
            continue
        if action == "quit":
            return
