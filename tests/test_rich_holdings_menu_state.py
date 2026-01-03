"""Tests for holdings menu selection state display."""

from datetime import datetime
from decimal import Decimal
from uuid import uuid4
from unittest.mock import MagicMock, patch

from rich.console import Console

from portfolio_manager.cli.rich_holdings import run_holdings_menu
from portfolio_manager.models import Account, Group, Holding, Stock


def test_run_holdings_menu_displays_current_account():
    """Should display current account in holdings menu."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.list_by_account.return_value = []

    run_holdings_menu(
        console, repo, account, prompt=lambda: "b", chooser=lambda **_: "back"
    )

    output = console.export_text()
    assert "Current Account" in output
    assert "Brokerage" in output


def test_run_holdings_menu_uses_group_name_in_selection_label():
    """Should include group name when rendering holding selection labels."""
    console = Console(record=True, width=80)
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    stock_id = uuid4()
    group_id = uuid4()
    holding = Holding(
        id=uuid4(),
        account_id=account.id,
        stock_id=stock_id,
        quantity=Decimal("2.0"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo = MagicMock()
    repo.list_by_account.return_value = [holding]

    stock_repo = MagicMock()
    stock_repo.get_by_id.return_value = Stock(
        id=stock_id,
        ticker="AAPL",
        group_id=group_id,
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    group_repo = MagicMock()
    group_repo.list_all.return_value = [
        Group(
            id=group_id,
            name="Tech",
            created_at=datetime.now(),
            updated_at=datetime.now(),
        )
    ]

    def _chooser(*_args, **kwargs):
        label_lookup = kwargs["label_lookup"]
        assert label_lookup(stock_id) == "Tech / AAPL"
        return None

    with patch(
        "portfolio_manager.cli.rich_holdings.choose_holding_menu",
        side_effect=["delete", "back"],
    ):
        with patch(
            "portfolio_manager.cli.rich_holdings.choose_holding_from_list",
            side_effect=_chooser,
        ):
            run_holdings_menu(
                console,
                repo,
                account,
                prompt=lambda: "b",
                stock_repository=stock_repo,
                group_repository=group_repo,
            )


def test_run_holdings_menu_edit_flow_invokes_update_holding():
    """Should call update flow when selecting edit in holdings menu."""
    console = Console(record=True, width=80)
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    holding = Holding(
        id=uuid4(),
        account_id=account.id,
        stock_id=uuid4(),
        quantity=Decimal("2.0"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo = MagicMock()
    repo.list_by_account.return_value = [holding]

    with patch(
        "portfolio_manager.cli.rich_holdings.choose_holding_menu",
        side_effect=["edit", "back"],
    ):
        with patch(
            "portfolio_manager.cli.rich_holdings.choose_holding_from_list",
            return_value=holding.id,
        ):
            with patch(
                "portfolio_manager.cli.rich_holdings.update_holding_flow",
            ) as update_holding_flow:
                run_holdings_menu(
                    console,
                    repo,
                    account,
                    prompt=lambda: "b",
                )

    update_holding_flow.assert_called_once()
