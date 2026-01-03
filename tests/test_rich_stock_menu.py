"""Tests for Rich-based stock menu flow."""

from datetime import datetime
from uuid import uuid4
from unittest.mock import MagicMock, patch

from rich.console import Console

from portfolio_manager.cli.rich_stocks import run_stock_menu
from portfolio_manager.models import Group, Stock


def test_run_stock_menu_renders_list_and_allows_back():
    """Should render stocks and exit on back command."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    group = Group(
        id=uuid4(),
        name="Tech Stocks",
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.list_by_group.return_value = []

    run_stock_menu(
        console,
        repo,
        group,
        prompt=lambda: "b",
        chooser=lambda **_: "back",
    )

    repo.list_by_group.assert_called_once_with(group.id)
    output = console.export_text()
    assert "Current Group" in output


def test_run_stock_menu_back_returns_to_group_menu():
    """Should return when selecting back in stock menu."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    group = Group(
        id=uuid4(),
        name="Tech Stocks",
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.list_by_group.return_value = []

    chooser = MagicMock(side_effect=["back"])

    run_stock_menu(
        console,
        repo,
        group,
        prompt=lambda: "b",
        chooser=chooser,
    )

    chooser.assert_called()


def test_run_stock_menu_add_flow_invokes_add_stock():
    """Should call add flow when selecting add in stock menu."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    group = Group(
        id=uuid4(),
        name="Tech Stocks",
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.list_by_group.return_value = []

    chooser = MagicMock(side_effect=["add", "back"])

    with patch(
        "portfolio_manager.cli.rich_stocks.add_stock_flow",
    ) as add_stock_flow:
        run_stock_menu(
            console,
            repo,
            group,
            prompt=lambda: "b",
            chooser=chooser,
        )

    add_stock_flow.assert_called_once()


def test_run_stock_menu_delete_flow_invokes_delete_stock():
    """Should call delete flow when selecting delete in stock menu."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    group = Group(
        id=uuid4(),
        name="Tech Stocks",
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    stock_id = uuid4()
    repo.list_by_group.return_value = []

    chooser = MagicMock(side_effect=["delete", "back"])

    with patch(
        "portfolio_manager.cli.rich_stocks.choose_stock_from_list",
        return_value=stock_id,
    ):
        with patch(
            "portfolio_manager.cli.rich_stocks.delete_stock_flow",
        ) as delete_stock_flow:
            run_stock_menu(
                console,
                repo,
                group,
                prompt=lambda: "b",
                chooser=chooser,
            )

    delete_stock_flow.assert_not_called()


def test_run_stock_menu_edit_flow_invokes_update_stock():
    """Should call update flow when selecting edit in stock menu."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    group = Group(
        id=uuid4(),
        name="Tech Stocks",
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    stock_id = uuid4()
    repo.list_by_group.return_value = []

    chooser = MagicMock(side_effect=["edit", "back"])

    with patch(
        "portfolio_manager.cli.rich_stocks.choose_stock_from_list",
        return_value=stock_id,
    ):
        with patch(
            "portfolio_manager.cli.rich_stocks.update_stock_flow",
        ) as update_stock_flow:
            run_stock_menu(
                console,
                repo,
                group,
                prompt=lambda: "b",
                chooser=chooser,
            )

    update_stock_flow.assert_not_called()


def test_run_stock_menu_move_flow_invokes_move_stock():
    """Should call move flow when selecting move in stock menu."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    group = Group(
        id=uuid4(),
        name="Tech Stocks",
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    stock_id = uuid4()
    target_group = Group(
        id=uuid4(),
        name="Growth",
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    stock = Stock(
        id=stock_id,
        ticker="AAPL",
        group_id=group.id,
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.list_by_group.return_value = [stock]
    group_repo = MagicMock()
    group_repo.list_all.return_value = [group, target_group]

    chooser = MagicMock(side_effect=["move", "back"])

    with patch(
        "portfolio_manager.cli.rich_stocks.choose_stock_from_list",
        return_value=stock_id,
    ):
        with patch(
            "portfolio_manager.cli.rich_stocks.choose_group_from_list",
            return_value=target_group.id,
        ):
            with patch(
                "portfolio_manager.cli.rich_stocks.move_stock_flow",
            ) as move_stock_flow:
                run_stock_menu(
                    console,
                    repo,
                    group,
                    prompt=lambda: "b",
                    chooser=chooser,
                    group_repository=group_repo,
                )

    move_stock_flow.assert_called_once()
