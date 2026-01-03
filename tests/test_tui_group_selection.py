"""Tests for group selection to show stocks."""

from datetime import datetime
from unittest.mock import MagicMock, Mock, patch
from uuid import uuid4

import pytest
from textual.widgets import ListView

from portfolio_manager.models import Group, Stock
from portfolio_manager.tui.app import PortfolioApp


@pytest.mark.asyncio
async def test_selecting_group_shows_stocks_for_that_group():
    """Should show stocks for selected group when Enter is pressed."""
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
                "portfolio_manager.tui.stock_screen.get_supabase_client"
            ) as mock_stock_client:
                with patch(
                    "portfolio_manager.tui.stock_screen.StockRepository"
                ) as mock_stock_repo_class:
                    # Arrange stocks
                    mock_stock_client.return_value = mock_client
                    mock_stock_repo = Mock()
                    mock_stock_repo_class.return_value = mock_stock_repo

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
                    mock_stock_repo.list_by_group.return_value = mock_stocks

                    async with app.run_test() as pilot:
                        # Act - go to groups screen
                        await pilot.click("#manage-groups-btn")
                        await pilot.pause()

                        # Select the first group and press Enter
                        list_view = pilot.app.screen.query_one("#groups-list", ListView)
                        list_view.index = 0
                        await pilot.press("enter")
                        await pilot.pause()

                        # Assert - should be on stock screen with stocks for that group
                        stocks_list = pilot.app.screen.query_one(
                            "#stocks-list", ListView
                        )
                        assert stocks_list is not None
                        assert len(stocks_list.children) == 2
                        mock_stock_repo.list_by_group.assert_called_with(group_id)
