"""Tests for TUI stock list screen."""

from datetime import datetime
from unittest.mock import Mock, patch, MagicMock
from uuid import uuid4

import pytest
from textual.widgets import ListView

from portfolio_manager.tui.app import PortfolioApp
from portfolio_manager.models import Group, Stock


@pytest.mark.asyncio
async def test_stock_list_screen_shows_list_view():
    """Should display a ListView for stocks when accessed through group selection."""
    app = PortfolioApp()

    with patch(
        "portfolio_manager.tui.group_screen.get_supabase_client"
    ) as mock_group_client:
        with patch(
            "portfolio_manager.tui.group_screen.GroupRepository"
        ) as mock_group_repo_class:
            # Arrange groups
            mock_client = MagicMock()
            mock_group_client.return_value = mock_client
            mock_group_repo = MagicMock()
            mock_group_repo_class.return_value = mock_group_repo

            group_id = uuid4()
            now = datetime.now()
            mock_groups = [
                Group(id=group_id, name="Tech Stocks", created_at=now, updated_at=now),
            ]
            mock_group_repo.list_all.return_value = mock_groups

            with patch(
                "portfolio_manager.tui.stock_screen.StockRepository"
            ) as MockRepository:
                with patch(
                    "portfolio_manager.tui.stock_screen.get_supabase_client"
                ) as mock_stock_client:
                    # Arrange stocks
                    mock_stock_client.return_value = mock_client
                    mock_repo = Mock()
                    MockRepository.return_value = mock_repo
                    mock_repo.list_by_group.return_value = []

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
                            pilot.app.screen.query_one("#stocks-list", ListView)
                            is not None
                        )


@pytest.mark.asyncio
async def test_stock_list_screen_loads_stocks_from_database():
    """Should load and display stocks from database when screen opens."""
    app = PortfolioApp()

    group_id = uuid4()
    now = datetime.now()

    with patch(
        "portfolio_manager.tui.group_screen.get_supabase_client"
    ) as mock_group_client:
        with patch(
            "portfolio_manager.tui.group_screen.GroupRepository"
        ) as mock_group_repo_class:
            # Arrange groups
            mock_client = MagicMock()
            mock_group_client.return_value = mock_client
            mock_group_repo = MagicMock()
            mock_group_repo_class.return_value = mock_group_repo

            mock_groups = [
                Group(id=group_id, name="Tech Stocks", created_at=now, updated_at=now),
            ]
            mock_group_repo.list_all.return_value = mock_groups

            with patch(
                "portfolio_manager.tui.stock_screen.StockRepository"
            ) as MockRepository:
                # Arrange stocks
                mock_repo = Mock()
                MockRepository.return_value = mock_repo

                # Mock stocks to return
                mock_stocks = [
                    Stock(
                        id=uuid4(),
                        ticker="AAPL",
                        group_id=group_id,
                        created_at=now,
                        updated_at=now,
                    ),
                    Stock(
                        id=uuid4(),
                        ticker="GOOGL",
                        group_id=group_id,
                        created_at=now,
                        updated_at=now,
                    ),
                ]
                mock_repo.list_by_group.return_value = mock_stocks

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
                        mock_repo.list_by_group.assert_called_once()
                        stock_list_view = pilot.app.screen.query_one(
                            "#stocks-list", ListView
                        )
                        assert len(stock_list_view.children) == 2
