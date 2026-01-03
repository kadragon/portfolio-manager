"""Tests for TUI add stock functionality."""

from datetime import datetime
from unittest.mock import Mock, patch, MagicMock
from uuid import uuid4

import pytest
from textual.widgets import Button, Input, ListView

from portfolio_manager.tui.app import PortfolioApp
from portfolio_manager.models import Group


@pytest.mark.asyncio
async def test_stock_list_screen_shows_add_button():
    """Should display a button to add a new stock."""
    app = PortfolioApp()

    group_id = uuid4()
    now = datetime.now()

    with patch(
        "portfolio_manager.tui.group_screen.get_supabase_client"
    ) as mock_group_client:
        with patch(
            "portfolio_manager.tui.group_screen.GroupRepository"
        ) as mock_group_repo_class:
            mock_client = MagicMock()
            mock_group_client.return_value = mock_client
            mock_group_repo = MagicMock()
            mock_group_repo_class.return_value = mock_group_repo

            mock_groups = [
                Group(id=group_id, name="Tech Stocks", created_at=now, updated_at=now)
            ]
            mock_group_repo.list_all.return_value = mock_groups

            with patch(
                "portfolio_manager.tui.stock_screen.StockRepository"
            ) as MockRepository:
                mock_repo = Mock()
                MockRepository.return_value = mock_repo
                mock_repo.list_by_group.return_value = []

                with patch(
                    "portfolio_manager.tui.stock_screen.get_supabase_client"
                ) as mock_stock_client:
                    mock_stock_client.return_value = mock_client

                    async with app.run_test() as pilot:
                        # Act - go to groups, select first group
                        await pilot.click("#manage-groups-btn")
                        await pilot.pause()
                        list_view = pilot.app.screen.query_one("#groups-list", ListView)
                        list_view.index = 0
                        await pilot.press("enter")
                        await pilot.pause()

                        # Assert
                        assert (
                            pilot.app.screen.query_one("#add-stock-btn", Button)
                            is not None
                        )


@pytest.mark.asyncio
async def test_add_stock_button_shows_input_field():
    """Should show input field when add stock button is clicked."""
    app = PortfolioApp()

    group_id = uuid4()
    now = datetime.now()

    with patch(
        "portfolio_manager.tui.group_screen.get_supabase_client"
    ) as mock_group_client:
        with patch(
            "portfolio_manager.tui.group_screen.GroupRepository"
        ) as mock_group_repo_class:
            mock_client = MagicMock()
            mock_group_client.return_value = mock_client
            mock_group_repo = MagicMock()
            mock_group_repo_class.return_value = mock_group_repo

            mock_groups = [
                Group(id=group_id, name="Tech Stocks", created_at=now, updated_at=now)
            ]
            mock_group_repo.list_all.return_value = mock_groups

            with patch(
                "portfolio_manager.tui.stock_screen.StockRepository"
            ) as MockRepository:
                mock_repo = Mock()
                MockRepository.return_value = mock_repo
                mock_repo.list_by_group.return_value = []

                with patch(
                    "portfolio_manager.tui.stock_screen.get_supabase_client"
                ) as mock_stock_client:
                    mock_stock_client.return_value = mock_client

                    async with app.run_test() as pilot:
                        # Act - go to groups, select first group, then add stock
                        await pilot.click("#manage-groups-btn")
                        await pilot.pause()
                        list_view = pilot.app.screen.query_one("#groups-list", ListView)
                        list_view.index = 0
                        await pilot.press("enter")
                        await pilot.pause()
                        await pilot.click("#add-stock-btn")
                        await pilot.pause()

                        # Assert
                        assert (
                            pilot.app.screen.query_one("#stock-ticker-input", Input)
                            is not None
                        )


@pytest.mark.asyncio
async def test_pressing_enter_saves_stock_to_database():
    """Should save stock to database when Enter is pressed in ticker input."""
    app = PortfolioApp()

    group_id = uuid4()
    now = datetime.now()

    with patch(
        "portfolio_manager.tui.group_screen.get_supabase_client"
    ) as mock_group_client:
        with patch(
            "portfolio_manager.tui.group_screen.GroupRepository"
        ) as mock_group_repo_class:
            mock_client = MagicMock()
            mock_group_client.return_value = mock_client
            mock_group_repo = MagicMock()
            mock_group_repo_class.return_value = mock_group_repo

            mock_groups = [
                Group(id=group_id, name="Tech Stocks", created_at=now, updated_at=now)
            ]
            mock_group_repo.list_all.return_value = mock_groups

            with patch(
                "portfolio_manager.tui.stock_screen.StockRepository"
            ) as MockRepository:
                mock_repo = Mock()
                MockRepository.return_value = mock_repo
                mock_repo.list_by_group.return_value = []

                with patch(
                    "portfolio_manager.tui.stock_screen.get_supabase_client"
                ) as mock_stock_client:
                    mock_stock_client.return_value = mock_client

                    async with app.run_test() as pilot:
                        # Act - go to groups, select first group, add stock
                        await pilot.click("#manage-groups-btn")
                        await pilot.pause()
                        list_view = pilot.app.screen.query_one("#groups-list", ListView)
                        list_view.index = 0
                        await pilot.press("enter")
                        await pilot.pause()
                        await pilot.click("#add-stock-btn")
                        await pilot.pause()

                        # Focus and submit the input
                        input_field = pilot.app.screen.query_one(
                            "#stock-ticker-input", Input
                        )
                        input_field.focus()
                        input_field.value = "AAPL"
                        await pilot.press("enter")
                        await pilot.pause()

                        # Assert
                        mock_repo.create.assert_called_once()
                        call_args = mock_repo.create.call_args
                        assert call_args[0][0] == "AAPL"  # ticker
