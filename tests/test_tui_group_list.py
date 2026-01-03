"""Tests for TUI group list screen."""

from unittest.mock import MagicMock, patch

import pytest
from textual.widgets import ListView

from portfolio_manager.tui.app import PortfolioApp


@pytest.mark.asyncio
async def test_group_list_screen_shows_list_view():
    """Should display a ListView for groups."""
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
            mock_repo.list_all.return_value = []

            async with app.run_test() as pilot:
                # Act
                await pilot.click("#manage-groups-btn")
                await pilot.pause()

                # Assert
                assert pilot.app.screen.query_one("#groups-list", ListView) is not None
