from textual.app import App, ComposeResult
from textual.widgets import Static


class PortfolioApp(App):
    """Terminal UI for portfolio management."""

    CSS = """
    Screen {
        align: center middle;
    }

    #app-title {
        padding: 1 2;
        border: heavy $accent;
    }
    """

    def compose(self) -> ComposeResult:
        yield Static("Portfolio Manager", id="app-title")
        yield Static("No symbols yet.", id="empty-state")
