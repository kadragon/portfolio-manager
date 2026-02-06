"""Tests for Rich-based holding flows."""

from datetime import datetime
from decimal import Decimal
from uuid import uuid4
from unittest.mock import MagicMock, patch

from rich.console import Console

from portfolio_manager.cli.holdings import (
    add_holding_flow,
    delete_holding_flow,
    render_holdings_for_account,
    run_holdings_menu,
    update_holding_flow,
)
from portfolio_manager.cli.prompt_select import (
    choose_holding_from_list,
    choose_holding_menu,
)
from portfolio_manager.models import Account, Group, Holding, Stock


def test_add_holding_flow_creates_holding_and_reports_quantity():
    """Should create holding and render confirmation."""
    console = Console(record=True, width=80)
    repo = MagicMock()
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
        quantity=Decimal("5.5"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.create.return_value = holding

    add_holding_flow(
        console,
        repo,
        account,
        prompt_stock=lambda: holding.stock_id,
        prompt_quantity=lambda: Decimal("5.5"),
    )

    repo.create.assert_called_once_with(
        account_id=account.id,
        stock_id=holding.stock_id,
        quantity=Decimal("5.5"),
    )
    output = console.export_text()
    assert "5.5" in output


def test_add_holding_flow_resolves_ticker_to_stock_id():
    """Should resolve ticker to stock id when adding holding."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    stock_repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    stock_id = uuid4()
    stock_repo.get_by_ticker.return_value = Stock(
        id=stock_id,
        ticker="310970",
        group_id=uuid4(),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    holding = Holding(
        id=uuid4(),
        account_id=account.id,
        stock_id=stock_id,
        quantity=Decimal("3"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.create.return_value = holding

    add_holding_flow(
        console,
        repo,
        account,
        prompt_stock=lambda: "310970",
        prompt_quantity=lambda: Decimal("3"),
        stock_repository=stock_repo,
    )

    stock_repo.get_by_ticker.assert_called_once_with("310970")
    repo.create.assert_called_once_with(
        account_id=account.id,
        stock_id=stock_id,
        quantity=Decimal("3"),
    )


def test_add_holding_flow_creates_stock_when_missing():
    """Should create a stock when ticker is missing."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    stock_repo = MagicMock()
    group_repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    group_id = uuid4()
    group_repo.list_all.return_value = [
        Group(
            id=group_id,
            name="Tech",
            created_at=datetime.now(),
            updated_at=datetime.now(),
        )
    ]
    stock_repo.get_by_ticker.return_value = None
    stock_repo.create.return_value = Stock(
        id=uuid4(),
        ticker="310970",
        group_id=group_id,
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    holding = Holding(
        id=uuid4(),
        account_id=account.id,
        stock_id=stock_repo.create.return_value.id,
        quantity=Decimal("2"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.create.return_value = holding

    add_holding_flow(
        console,
        repo,
        account,
        prompt_stock=lambda: "310970",
        prompt_quantity=lambda: Decimal("2"),
        stock_repository=stock_repo,
        group_repository=group_repo,
        group_chooser=lambda **_: group_id,
    )

    stock_repo.create.assert_called_once_with("310970", group_id)
    repo.create.assert_called_once_with(
        account_id=account.id,
        stock_id=stock_repo.create.return_value.id,
        quantity=Decimal("2"),
    )


def test_add_holding_flow_creates_group_when_missing():
    """Should create a group when none exist."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    stock_repo = MagicMock()
    group_repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    group_repo.list_all.return_value = []
    group_repo.create.return_value = Group(
        id=uuid4(),
        name="Auto",
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    stock_repo.get_by_ticker.return_value = None
    stock_repo.create.return_value = Stock(
        id=uuid4(),
        ticker="310970",
        group_id=group_repo.create.return_value.id,
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.create.return_value = Holding(
        id=uuid4(),
        account_id=account.id,
        stock_id=stock_repo.create.return_value.id,
        quantity=Decimal("2"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )

    add_holding_flow(
        console,
        repo,
        account,
        prompt_stock=lambda: "310970",
        prompt_quantity=lambda: Decimal("2"),
        stock_repository=stock_repo,
        group_repository=group_repo,
        group_chooser=lambda **_: None,
        prompt_group_name=lambda: "Auto",
    )

    group_repo.create.assert_called_once_with("Auto")
    stock_repo.create.assert_called_once_with(
        "310970", group_repo.create.return_value.id
    )


def test_delete_holding_flow_removes_holding_and_reports_quantity():
    """Should delete holding and render confirmation."""
    console = Console(record=True, width=80)
    repo = MagicMock()
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
        quantity=Decimal("5.5"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )

    delete_holding_flow(console, repo, account, holding, confirm=lambda: True)

    repo.delete.assert_called_once_with(holding.id)
    output = console.export_text()
    assert "5.5" in output


def test_render_holdings_for_account_lists_holdings():
    """Should render holdings for the selected account."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    holdings = [
        Holding(
            id=uuid4(),
            account_id=account.id,
            stock_id=uuid4(),
            quantity=Decimal("5.5"),
            created_at=datetime.now(),
            updated_at=datetime.now(),
        ),
        Holding(
            id=uuid4(),
            account_id=account.id,
            stock_id=uuid4(),
            quantity=Decimal("10"),
            created_at=datetime.now(),
            updated_at=datetime.now(),
        ),
    ]
    repo.list_by_account.return_value = holdings

    render_holdings_for_account(console, repo, account)

    repo.list_by_account.assert_called_once_with(account.id)
    output = console.export_text()
    assert "5.5" in output
    assert "10" in output


def test_render_holdings_for_account_displays_ticker():
    """Should render ticker instead of raw stock ID when mapping provided."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    stock_id = uuid4()
    holdings = [
        Holding(
            id=uuid4(),
            account_id=account.id,
            stock_id=stock_id,
            quantity=Decimal("5.5"),
            created_at=datetime.now(),
            updated_at=datetime.now(),
        )
    ]
    repo.list_by_account.return_value = holdings

    render_holdings_for_account(
        console, repo, account, stock_lookup=lambda _stock_id: "AAPL"
    )

    output = console.export_text()
    assert "AAPL" in output


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
        "portfolio_manager.cli.holdings.choose_holding_menu",
        side_effect=["delete", "back"],
    ):
        with patch(
            "portfolio_manager.cli.holdings.choose_holding_from_list",
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
        "portfolio_manager.cli.holdings.choose_holding_menu",
        side_effect=["edit", "back"],
    ):
        with patch(
            "portfolio_manager.cli.holdings.choose_holding_from_list",
            return_value=holding.id,
        ):
            with patch(
                "portfolio_manager.cli.holdings.update_holding_flow",
            ) as update_holding_flow:
                run_holdings_menu(
                    console,
                    repo,
                    account,
                    prompt=lambda: "b",
                )

    update_holding_flow.assert_called_once()


def test_choose_holding_menu_returns_selected_action():
    """Should return the selected action from chooser."""
    chooser = MagicMock(return_value="delete")

    action = choose_holding_menu(chooser)

    chooser.assert_called_once()
    assert action == "delete"


def test_choose_holding_from_list_returns_holding_id():
    """Should return the selected holding id."""
    holding_id = uuid4()
    holdings = [
        Holding(
            id=holding_id,
            account_id=uuid4(),
            stock_id=uuid4(),
            quantity=Decimal("1.5"),
            created_at=datetime.now(),
            updated_at=datetime.now(),
        ),
    ]
    chooser = MagicMock(return_value=holding_id)

    result = choose_holding_from_list(holdings, chooser)

    chooser.assert_called_once()
    assert result == holding_id


def test_choose_holding_from_list_uses_stock_label_lookup():
    """Should render holding labels using provided stock lookup."""
    holding_id = uuid4()
    stock_id = uuid4()
    holdings = [
        Holding(
            id=holding_id,
            account_id=uuid4(),
            stock_id=stock_id,
            quantity=Decimal("2.0"),
            created_at=datetime.now(),
            updated_at=datetime.now(),
        ),
    ]
    chooser = MagicMock(return_value=holding_id)

    choose_holding_from_list(
        holdings,
        chooser=chooser,
        label_lookup=lambda _stock_id: "AAPL",
    )

    _, kwargs = chooser.call_args
    options = kwargs["options"]
    assert options[0][0] == holding_id
    assert options[0][1] == "AAPL (2.0)"


def test_add_holding_flow_cancelled_does_not_create():
    """Should not create holding when user cancels stock input."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    account = Account(
        id=uuid4(),
        name="Brokerage",
        cash_balance=Decimal("1000.25"),
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )

    add_holding_flow(
        console,
        repo,
        account,
        prompt_stock=lambda: None,
        prompt_quantity=lambda: Decimal("5.5"),
    )

    repo.create.assert_not_called()
    output = console.export_text()
    assert "Cancelled" in output


def test_update_holding_flow_blank_quantity_keeps_existing_value():
    """Blank quantity input should keep the current quantity."""
    console = Console(record=True, width=80)
    repo = MagicMock()
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
    repo.update.return_value = holding

    update_holding_flow(
        console,
        repo,
        account,
        holding,
        prompt_quantity=lambda: "   ",
    )

    repo.update.assert_called_once_with(holding.id, quantity=Decimal("2.0"))


def test_update_holding_flow_invalid_quantity_keeps_existing_value():
    """Invalid quantity input should keep the current quantity."""
    console = Console(record=True, width=80)
    repo = MagicMock()
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
    repo.update.return_value = holding

    update_holding_flow(
        console,
        repo,
        account,
        holding,
        prompt_quantity=lambda: "oops",
    )

    repo.update.assert_called_once_with(holding.id, quantity=Decimal("2.0"))
    output = console.export_text()
    assert "Invalid quantity" in output
