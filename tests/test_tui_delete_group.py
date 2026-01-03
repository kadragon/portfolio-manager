"""Tests for deleting groups from TUI."""

from datetime import datetime
from unittest.mock import MagicMock, patch
from uuid import uuid4

import pytest
from textual.widgets import ListView

from portfolio_manager.models import Group
from portfolio_manager.tui.app import PortfolioApp


@pytest.mark.asyncio
async def test_pressing_delete_key_removes_selected_group():
    """Should delete selected group when Delete key is pressed."""
    app = PortfolioApp()

    with patch(
        "portfolio_manager.tui.group_screen.get_supabase_client"
    ) as mock_client_factory:
        with patch(
            "portfolio_manager.tui.group_screen.GroupRepository"
        ) as mock_repo_class:
            # Arrange
            mock_client = MagicMock()
            mock_client_factory.return_value = mock_client
            mock_repo = MagicMock()
            mock_repo_class.return_value = mock_repo

            # Mock groups data
            group1 = Group(
                id=uuid4(),
                name="Tech Stocks",
                created_at=datetime.now(),
                updated_at=datetime.now(),
            )
            group2 = Group(
                id=uuid4(),
                name="Blue Chips",
                created_at=datetime.now(),
                updated_at=datetime.now(),
            )
            mock_repo.list_all.return_value = [group1, group2]

            async with app.run_test() as pilot:
                # Act
                await pilot.click("#manage-groups-btn")
                await pilot.pause()

                # Select first group and press Delete
                list_view = pilot.app.screen.query_one("#groups-list", ListView)
                list_view.index = 0
                await pilot.press("delete")
                await pilot.pause()

                # Assert
                mock_repo.delete.assert_called_once_with(group1.id)
