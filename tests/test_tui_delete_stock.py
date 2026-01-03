"""Tests for TUI delete stock functionality."""

from datetime import datetime
from unittest.mock import Mock, patch, MagicMock
from uuid import uuid4

import pytest
from textual.widgets import ListView

from portfolio_manager.tui.app import PortfolioApp
from portfolio_manager.models import Group, Stock


@pytest.mark.asyncio
async def test_pressing_delete_key_removes_stock():
    """Should delete stock when Delete key is pressed on a selected item."""
    app = PortfolioApp()

    group_id = uuid4()
    stock_id = uuid4()
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
                # Arrange
                mock_repo = Mock()
                MockRepository.return_value = mock_repo

                # Mock stocks to return
                mock_stocks = [
                    Stock(
                        id=stock_id,
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

                        # Select first stock and press Delete
                        stock_list_view = pilot.app.screen.query_one(
                            "#stocks-list", ListView
                        )
                        stock_list_view.index = 0
                        await pilot.press("delete")
                        await pilot.pause()

                        # Assert
                        mock_repo.delete.assert_called_once_with(stock_id)
