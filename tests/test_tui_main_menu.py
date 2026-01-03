"""Tests for TUI main menu."""

import pytest
from textual.widgets import Button

from portfolio_manager.tui.app import PortfolioApp


@pytest.mark.asyncio
async def test_main_menu_shows_groups_button():
    """Should display a button to manage groups."""
    app = PortfolioApp()
    async with app.run_test() as pilot:
        # Assert
        assert pilot.app.query_one("#manage-groups-btn", Button)
