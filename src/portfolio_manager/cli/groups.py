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
    table.add_column("Target %", justify="right")
    for index, group in enumerate(groups, start=1):
        table.add_row(str(index), group.name, f"{group.target_percentage:.1f}%")
    console.print(table)


def add_group_flow(
    console: Console, repository, prompt: Callable[[], str] | None = None
) -> None:
    """Add a group via prompt and render confirmation."""
    if prompt:
        name = prompt()
        target_str = prompt()
    else:
        name = Prompt.ask("Group name")
        target_str = Prompt.ask("Target Percentage", default="0.0")

    try:
        target_percentage = float(target_str)
    except ValueError:
        console.print("[yellow]Invalid percentage, defaulting to 0.0[/yellow]")
        target_percentage = 0.0

    group = repository.create(name, target_percentage=target_percentage)
    console.print(f"Added group: {group.name} (Target: {group.target_percentage}%)")


def update_group_flow(
    console: Console,
    repository,
    group: Group,
    prompt: Callable[[], str] | None = None,
) -> None:
    """Update a group name via prompt and render confirmation."""
    current_target = group.target_percentage

    if prompt:
        name = prompt()
        target_str = prompt()
    else:
        name = Prompt.ask("New group name", default=group.name)
        target_str = Prompt.ask("New target percentage", default=str(current_target))

    try:
        target_percentage = float(target_str)
    except ValueError:
        console.print("[yellow]Invalid percentage, keeping current value[/yellow]")
        target_percentage = current_target

    updated = repository.update(
        group.id, name=name, target_percentage=target_percentage
    )
    console.print(
        f"Updated group: {updated.name} (Target: {updated.target_percentage}%)"
    )


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
