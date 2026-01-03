"""Shared Rich menu helpers."""

from rich.console import Console


def render_menu_options(console: Console, options: str) -> None:
    """Render a standardized options line."""
    console.print(f"[bold]Options:[/bold] {options}")
