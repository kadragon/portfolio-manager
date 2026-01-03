"""Rich-only CLI entrypoint."""

from __future__ import annotations

from dataclasses import dataclass
from typing import Iterable

from dotenv import load_dotenv
from rich.console import Console
from rich.prompt import Prompt

from portfolio_manager.cli.rich_app import render_main_menu, select_main_menu_option
from portfolio_manager.cli.rich_groups import (
    add_group_flow,
    delete_group_flow,
    render_group_list,
    select_group_menu_option,
)
from portfolio_manager.cli.rich_stocks import run_stock_menu
from portfolio_manager.models import Group
from portfolio_manager.repositories.group_repository import GroupRepository
from portfolio_manager.repositories.stock_repository import StockRepository
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


def run_group_menu(console: Console) -> None:
    """Run the group management menu loop."""
    while True:
        groups = _load_groups()
        render_group_list(console, groups)
        console.print("[bold]Options:[/bold] a=add, d=delete, b=back, <number>=select")
        choice = Prompt.ask("Group menu")
        action = select_group_menu_option(choice)
        if action == "back":
            return
        if choice.strip().lower() == "a":
            client = get_supabase_client()
            repo = GroupRepository(client)
            add_group_flow(console, repo)
            continue
        if choice.strip().lower() == "d":
            index_text = Prompt.ask("Group number to delete")
            if index_text.isdigit():
                group = _select_group_by_index(groups, int(index_text))
                if group is not None:
                    client = get_supabase_client()
                    repo = GroupRepository(client)
                    delete_group_flow(console, repo, group)
            continue
        if choice.strip().isdigit():
            group = _select_group_by_index(groups, int(choice.strip()))
            if group is not None:
                client = get_supabase_client()
                repo = StockRepository(client)
                run_stock_menu(
                    console, repo, group, prompt=lambda: Prompt.ask("Stock menu")
                )
            continue


def main() -> None:
    """CLI entrypoint."""
    load_dotenv()
    console = Console()
    while True:
        render_main_menu(console)
        console.print("[bold]Options:[/bold] g=groups, q=quit")
        choice = Prompt.ask("Select")
        action = select_main_menu_option(choice)
        if action == "groups":
            run_group_menu(console)
            continue
        if choice.strip().lower() == "q":
            return
