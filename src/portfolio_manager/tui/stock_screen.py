"""Stock management screen."""

from uuid import UUID
from textual import events
from textual.app import ComposeResult
from textual.containers import Container, Vertical
from textual.screen import Screen
from textual.widgets import Button, Input, ListView, ListItem, Label

from portfolio_manager.models import Stock
from portfolio_manager.repositories.stock_repository import StockRepository
from portfolio_manager.services.supabase_client import get_supabase_client


class StockListScreen(Screen):
    """Screen for managing stocks."""

    def __init__(self, group_id: UUID | None = None):
        """Initialize the screen.

        Args:
            group_id: The ID of the group to show stocks for. If None, uses hardcoded default.
        """
        super().__init__()
        self.stocks: list[Stock] = []
        self.group_id = (
            group_id
            if group_id is not None
            else UUID("00000000-0000-0000-0000-000000000001")
        )

    def compose(self) -> ComposeResult:
        yield Vertical(
            ListView(id="stocks-list"),
            Button("Add Stock", id="add-stock-btn"),
            Container(id="input-container"),
        )

    def on_mount(self) -> None:
        """Load stocks from database when screen is mounted."""
        client = get_supabase_client()
        repository = StockRepository(client)
        self.stocks = repository.list_by_group(self.group_id)

        list_view = self.query_one("#stocks-list", ListView)
        for stock in self.stocks:
            list_view.append(ListItem(Label(stock.ticker)))

    def on_button_pressed(self, event: Button.Pressed) -> None:
        """Handle button press events."""
        if event.button.id == "add-stock-btn":
            container = self.query_one("#input-container")
            container.mount(Input(placeholder="Stock ticker", id="stock-ticker-input"))

    def on_input_submitted(self, event: Input.Submitted) -> None:
        """Handle input submission (Enter key press)."""
        if event.input.id == "stock-ticker-input":
            ticker = event.value.strip()
            if ticker:
                client = get_supabase_client()
                repository = StockRepository(client)
                repository.create(ticker, self.group_id)

    def on_key(self, event: events.Key) -> None:
        """Handle key press events."""
        if event.key == "delete":
            list_view = self.query_one("#stocks-list", ListView)
            if list_view.index is not None and 0 <= list_view.index < len(self.stocks):
                stock_to_delete = self.stocks[list_view.index]

                client = get_supabase_client()
                repository = StockRepository(client)
                repository.delete(stock_to_delete.id)
