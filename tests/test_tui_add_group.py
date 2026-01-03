"""Tests for TUI add group functionality."""

from unittest.mock import MagicMock, patch
import pytest
from textual.widgets import Button, Input

from portfolio_manager.tui.app import PortfolioApp


@pytest.mark.asyncio
async def test_group_list_screen_shows_add_button():
    """Should display a button to add a new group."""
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
                assert pilot.app.screen.query_one("#add-group-btn", Button) is not None


@pytest.mark.asyncio
async def test_add_group_button_shows_input_field():
    """Should show input field when add group button is clicked."""
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
                await pilot.click("#add-group-btn")
                await pilot.pause()

                # Assert
                assert (
                    pilot.app.screen.query_one("#group-name-input", Input) is not None
                )


@pytest.mark.asyncio
async def test_pressing_enter_saves_group_to_database():
    """Should save group to database when Enter is pressed in input field."""
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
                await pilot.click("#add-group-btn")
                await pilot.pause()

                input_field = pilot.app.screen.query_one("#group-name-input", Input)
                input_field.focus()
                input_field.value = "Tech Stocks"
                await pilot.press("enter")
                await pilot.pause()

                # Assert
                mock_repo.create.assert_called_once_with("Tech Stocks")
