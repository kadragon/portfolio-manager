"""Rich-based group list rendering."""

from typing import Callable

from rich.console import Console
from rich.table import Table
from rich.prompt import Confirm, Prompt

from portfolio_manager.models import Group


def render_group_list(console: Console, groups: list[Group]) -> None:
    """Render the group list or an empty-state message."""
    if not groups:
        console.print("No groups found")
        return
    table = Table(title="Groups", header_style="bold")
    table.add_column("#", style="dim", width=4, justify="right")
    table.add_column("Name", style="bold")
    for index, group in enumerate(groups, start=1):
        table.add_row(str(index), group.name)
    console.print(table)


def add_group_flow(
    console: Console, repository, prompt: Callable[[], str] | None = None
) -> None:
    """Add a group via prompt and render confirmation."""
    prompt_func = prompt or (lambda: Prompt.ask("Group name"))
    name = prompt_func()
    group = repository.create(name)
    console.print(f"Added group: {group.name}")


def delete_group_flow(
    console: Console,
    repository,
    group: Group,
    confirm: Callable[[], bool] | None = None,
) -> None:
    """Delete a group with confirmation and render status."""
    confirm_func = confirm or (
        lambda: Confirm.ask(f"Delete {group.name}?", default=False)
    )
    if not confirm_func():
        return
    repository.delete(group.id)
    console.print(f"Deleted group: {group.name}")


def select_group_menu_option(choice: str) -> str | None:
    """Map a group menu choice to an action."""
    normalized = choice.strip().lower()
    if normalized == "b":
        return "back"
    return None
