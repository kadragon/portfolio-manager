"""prompt_toolkit-based menu selection helpers."""

from typing import Callable, Iterable
from uuid import UUID

from portfolio_manager.models import Account, Group, Holding, Stock


OptionList = Iterable[tuple[str, str]]


def choose_main_menu(chooser: Callable | None = None) -> str | None:
    """Choose a main menu action using prompt_toolkit."""
    if chooser is None:
        from prompt_toolkit.shortcuts import choice

        chooser = choice

    options: OptionList = [
        ("groups", "Groups"),
        ("accounts", "Accounts"),
        ("quit", "Quit"),
    ]
    return chooser(message="Select menu:", options=options, default="groups")


def choose_group_menu(chooser: Callable | None = None) -> str | None:
    """Choose a group menu action using prompt_toolkit."""
    if chooser is None:
        from prompt_toolkit.shortcuts import choice

        chooser = choice

    options: OptionList = [
        ("add", "Add group"),
        ("edit", "Edit group"),
        ("delete", "Delete group"),
        ("select", "Select group"),
        ("back", "Back"),
    ]
    return chooser(message="Group menu:", options=options, default="select")


def choose_stock_menu(chooser: Callable | None = None) -> str | None:
    """Choose a stock menu action using prompt_toolkit."""
    if chooser is None:
        from prompt_toolkit.shortcuts import choice

        chooser = choice

    options: OptionList = [
        ("add", "Add stock"),
        ("edit", "Edit stock"),
        ("move", "Move stock"),
        ("delete", "Delete stock"),
        ("back", "Back"),
    ]
    return chooser(message="Stock menu:", options=options, default="back")


def choose_account_menu(chooser: Callable | None = None) -> str | None:
    """Choose an account menu action using prompt_toolkit."""
    if chooser is None:
        from prompt_toolkit.shortcuts import choice

        chooser = choice

    options: OptionList = [
        ("add", "Add account"),
        ("edit", "Edit account"),
        ("delete", "Delete account"),
        ("select", "Select account"),
        ("back", "Back"),
    ]
    return chooser(message="Accounts menu:", options=options, default="select")


def choose_holding_menu(chooser: Callable | None = None) -> str | None:
    """Choose a holding menu action using prompt_toolkit."""
    if chooser is None:
        from prompt_toolkit.shortcuts import choice

        chooser = choice

    options: OptionList = [
        ("add", "Add holding"),
        ("edit", "Edit holding"),
        ("delete", "Delete holding"),
        ("back", "Back"),
    ]
    return chooser(message="Holdings menu:", options=options, default="back")


def choose_group_from_list(
    groups: list[Group], chooser: Callable | None = None
) -> UUID | None:
    """Choose a group from a list using prompt_toolkit."""
    if chooser is None:
        from prompt_toolkit.shortcuts import choice

        chooser = choice
    options = [(group.id, group.name) for group in groups]
    if not options:
        return None
    return chooser(message="Select group:", options=options, default=options[0][0])


def choose_account_from_list(
    accounts: list[Account], chooser: Callable | None = None
) -> UUID | None:
    """Choose an account from a list using prompt_toolkit."""
    if chooser is None:
        from prompt_toolkit.shortcuts import choice

        chooser = choice
    options = [(account.id, account.name) for account in accounts]
    if not options:
        return None
    return chooser(message="Select account:", options=options, default=options[0][0])


def choose_stock_from_list(
    stocks: list[Stock], chooser: Callable | None = None
) -> UUID | None:
    """Choose a stock from a list using prompt_toolkit."""
    if chooser is None:
        from prompt_toolkit.shortcuts import choice

        chooser = choice
    options = [(stock.id, stock.ticker) for stock in stocks]
    if not options:
        return None
    return chooser(message="Select stock:", options=options, default=options[0][0])


def choose_holding_from_list(
    holdings: list[Holding],
    chooser: Callable | None = None,
    label_lookup: Callable[[UUID], str] | None = None,
) -> UUID | None:
    """Choose a holding from a list using prompt_toolkit."""
    if chooser is None:
        from prompt_toolkit.shortcuts import choice

        chooser = choice
    lookup = label_lookup or (lambda stock_id: str(stock_id))
    options = [
        (holding.id, f"{lookup(holding.stock_id)} ({holding.quantity})")
        for holding in holdings
    ]
    if not options:
        return None
    return chooser(message="Select holding:", options=options, default=options[0][0])
