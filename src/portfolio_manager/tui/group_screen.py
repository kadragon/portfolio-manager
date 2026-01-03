"""Group management screen."""

from textual.app import ComposeResult
from textual.containers import Container, Vertical
from textual.screen import Screen
from textual.widgets import Button, Input, Label, ListItem, ListView

from portfolio_manager.repositories.group_repository import GroupRepository
from portfolio_manager.services.supabase_client import get_supabase_client
from portfolio_manager.tui.stock_screen import StockListScreen


class GroupListScreen(Screen):
    """Screen for managing groups."""

    def __init__(self, *args, **kwargs):
        """Initialize the screen."""
        super().__init__(*args, **kwargs)
        self.groups = []

    def compose(self) -> ComposeResult:
        yield Vertical(
            ListView(id="groups-list"),
            Button("Add Group", id="add-group-btn"),
            Container(id="input-container"),
        )

    def on_mount(self) -> None:
        """Load groups from database when screen is mounted."""
        client = get_supabase_client()
        repo = GroupRepository(client)
        self.groups = repo.list_all()

        list_view = self.query_one("#groups-list", ListView)
        for group in self.groups:
            list_view.append(ListItem(Label(group.name)))

    def on_button_pressed(self, event: Button.Pressed) -> None:
        """Handle button press events."""
        if event.button.id == "add-group-btn":
            container = self.query_one("#input-container")
            container.mount(Input(placeholder="Group name", id="group-name-input"))

    def on_input_submitted(self, event: Input.Submitted) -> None:
        """Handle input submission events."""
        if event.input.id == "group-name-input":
            group_name = event.input.value.strip()
            if group_name:
                client = get_supabase_client()
                repo = GroupRepository(client)
                repo.create(group_name)

    def on_key(self, event) -> None:
        """Handle key press events."""
        if event.key == "delete":
            list_view = self.query_one("#groups-list", ListView)
            if list_view.index is not None and 0 <= list_view.index < len(self.groups):
                group_to_delete = self.groups[list_view.index]
                client = get_supabase_client()
                repo = GroupRepository(client)
                repo.delete(group_to_delete.id)
        elif event.key == "enter":
            list_view = self.query_one("#groups-list", ListView)
            if list_view.index is not None and 0 <= list_view.index < len(self.groups):
                selected_group = self.groups[list_view.index]
                self.app.push_screen(StockListScreen(group_id=selected_group.id))
