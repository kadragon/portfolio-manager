import pytest
from textual.widgets import Static

from portfolio_manager.tui.app import PortfolioApp


@pytest.mark.asyncio
async def test_app_renders_header_title():
    app = PortfolioApp()

    async with app.run_test():
        header = app.query_one("#app-title", Static)
        assert header is not None
        assert "portfolio" in header.render().plain.lower()  # type: ignore
