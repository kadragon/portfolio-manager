"""prompt_toolkit-based menu selection helpers."""

from decimal import Decimal, InvalidOperation
from typing import Callable, Iterable
from uuid import UUID

from prompt_toolkit import PromptSession
from prompt_toolkit.key_binding import KeyBindings
from rich.console import Console

from portfolio_manager.models import Account, Deposit, Group, Holding, Stock


def _create_esc_bindings() -> KeyBindings:
    """Create key bindings that abort on ESC."""
    bindings = KeyBindings()

    @bindings.add("escape")
    def _(event):
        event.app.exit(exception=EOFError())

    return bindings


def cancellable_prompt(
    message: str,
    *,
    default: str = "",
    session: PromptSession | None = None,
) -> str | None:
    """Prompt for input with ESC and Ctrl+C cancellation support.

    Returns None if user cancels (ESC, Ctrl+C, Ctrl+D).
    """
    if session is None:
        session = PromptSession(key_bindings=_create_esc_bindings())

    try:
        return session.prompt(f"{message} ", default=default)
    except (KeyboardInterrupt, EOFError):
        return None


def prompt_decimal(
    message: str,
    default: str = "",
    session: PromptSession | None = None,
    console: Console | None = None,
) -> Decimal | None:
    """Prompt for a decimal value with cancellation support."""
    while True:
        value = cancellable_prompt(message, default=default, session=session)
        if value is None:
            return None
        try:
            return Decimal(value)
        except InvalidOperation:
            if console is not None:
                console.print("[red]Invalid number. Please try again.[/red]")
            else:
                print("Invalid number. Please try again.")
            continue


OptionList = Iterable[tuple[str, str]]


def choose_main_menu(chooser: Callable | None = None) -> str | None:
    """Choose a main menu action using prompt_toolkit."""
    if chooser is None:
        from prompt_toolkit.shortcuts import choice

        chooser = choice

    options: OptionList = [
        ("groups", "Groups"),
        ("accounts", "Accounts"),
        ("deposits", "Deposits"),
        ("rebalance", "Rebalance"),
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
        ("quick", "Quick update cash"),
        ("sync", "Sync KIS account"),
        ("add", "Add account"),
        ("edit", "Edit account"),
        ("delete", "Delete account"),
        ("select", "Select account"),
        ("back", "Back"),
    ]
    return chooser(message="Accounts menu:", options=options, default="quick")


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


def choose_deposit_menu(chooser: Callable | None = None) -> str | None:
    """Choose a deposit menu action using prompt_toolkit."""
    if chooser is None:
        from prompt_toolkit.shortcuts import choice

        chooser = choice

    options: OptionList = [
        ("add", "Add deposit"),
        ("edit", "Edit deposit"),
        ("delete", "Delete deposit"),
        ("back", "Back"),
    ]
    return chooser(message="Deposits menu:", options=options, default="back")


def choose_rebalance_action(chooser: Callable | None = None) -> str | None:
    """Choose a rebalance action using prompt_toolkit."""
    if chooser is None:
        from prompt_toolkit.shortcuts import choice

        chooser = choice

    options: OptionList = [
        ("preview", "Preview only"),
        ("execute", "Execute orders"),
    ]
    return chooser(message="Rebalance action:", options=options, default="preview")


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


def choose_deposit_from_list(
    deposits: list[Deposit], chooser: Callable | None = None
) -> UUID | None:
    """Choose a deposit from a list using prompt_toolkit."""
    if chooser is None:
        from prompt_toolkit.shortcuts import choice

        chooser = choice
    options = [
        (deposit.id, f"{deposit.deposit_date.isoformat()} / {deposit.amount:,.0f}")
        for deposit in deposits
    ]
    if not options:
        return None
    return chooser(message="Select deposit:", options=options, default=options[0][0])
