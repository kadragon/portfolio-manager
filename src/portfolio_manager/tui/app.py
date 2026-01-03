from textual.app import App, ComposeResult
from textual.widgets import Button, Static

from portfolio_manager.tui.group_screen import GroupListScreen


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
        yield Static("Select a group to manage stocks.", id="empty-state")
        yield Button("Manage Groups", id="manage-groups-btn")

    def on_button_pressed(self, event: Button.Pressed) -> None:
        """Handle button press events."""
        if event.button.id == "manage-groups-btn":
            self.push_screen(GroupListScreen())


if __name__ == "__main__":
    from dotenv import load_dotenv

    load_dotenv()

    app = PortfolioApp()
    app.run()
