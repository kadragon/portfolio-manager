import pytest
from textual.widgets import Static

from portfolio_manager.tui.app import PortfolioApp


@pytest.mark.asyncio
async def test_empty_list_view_shows_placeholder():
    app = PortfolioApp()

    async with app.run_test():
        empty_state = app.query_one("#empty-state", Static)
        assert empty_state is not None
        assert "group" in empty_state.render().plain.lower()  # type: ignore
