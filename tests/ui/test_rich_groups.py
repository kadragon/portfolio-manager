"""Tests for Rich-based group flows."""

from datetime import datetime
from uuid import uuid4
from unittest.mock import MagicMock, patch

from rich.console import Console

from portfolio_manager.cli.groups import (
    add_group_flow,
    delete_group_flow,
    render_group_list,
    update_group_flow,
)
from portfolio_manager.cli import main as main_app
from portfolio_manager.models import Group
from portfolio_manager.cli.prompt_select import (
    choose_group_from_list,
    choose_group_menu,
)


def test_add_group_flow_creates_group_and_reports_name():
    """Should create a group and render confirmation."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    group = Group(
        id=uuid4(),
        name="Tech Stocks",
        target_percentage=10.0,
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.create.return_value = group

    # Mock prompt to return name then percentage
    prompt_mock = MagicMock(side_effect=["Tech Stocks", "10.0"])

    add_group_flow(console, repo, prompt=prompt_mock)

    repo.create.assert_called_once_with("Tech Stocks", target_percentage=10.0)
    output = console.export_text()
    assert "Tech Stocks" in output
    assert "10.0%" in output


def test_render_group_list_shows_target_percentage():
    """Should show target percentage in group list."""
    console = Console(record=True, width=80)
    group = Group(
        id=uuid4(),
        name="Tech Stocks",
        target_percentage=12.5,
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )

    render_group_list(console, [group])

    output = console.export_text()
    assert "Tech Stocks" in output
    assert "12.5" in output


def test_delete_group_flow_removes_group_and_reports_name():
    """Should delete group and render confirmation."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    group = Group(
        id=uuid4(),
        name="Tech Stocks",
        target_percentage=0.0,
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )

    delete_group_flow(console, repo, group, confirm=lambda: True)

    repo.delete.assert_called_once_with(group.id)
    output = console.export_text()
    assert "Tech Stocks" in output


def test_render_group_list_shows_empty_message_when_no_groups():
    """Should show empty message when no groups exist."""
    console = Console(record=True, width=80)

    render_group_list(console, [])

    output = console.export_text()
    assert "No groups found" in output


def test_run_group_menu_add_flow_invokes_add_group():
    """Should call add flow when selecting add in group menu."""
    console = Console(record=True, width=80)
    container = MagicMock()

    with patch.object(main_app, "_load_groups", return_value=[]):
        with patch.object(main_app, "choose_group_menu", side_effect=["add", "back"]):
            with patch.object(main_app, "add_group_flow") as add_group_flow:
                main_app.run_group_menu(console, container)

    add_group_flow.assert_called_once_with(console, container.group_repository)


def test_run_group_menu_delete_flow_invokes_delete_group():
    """Should call delete flow when selecting delete in group menu."""
    console = Console(record=True, width=80)
    container = MagicMock()

    with patch.object(main_app, "_load_groups", return_value=[]):
        with patch.object(
            main_app, "choose_group_menu", side_effect=["delete", "back"]
        ):
            with patch.object(main_app, "choose_group_from_list", return_value=None):
                with patch.object(main_app, "delete_group_flow") as delete_group_flow:
                    main_app.run_group_menu(console, container)

    delete_group_flow.assert_not_called()


def test_run_group_menu_select_flow_invokes_stock_menu():
    """Should open stock menu when selecting a group."""
    console = Console(record=True, width=80)
    container = MagicMock()
    group = main_app.Group(
        id=uuid4(),
        name="Tech",
        target_percentage=0.0,
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )

    with patch.object(main_app, "_load_groups", return_value=[group]):
        with patch.object(
            main_app, "choose_group_menu", side_effect=["select", "back"]
        ):
            with patch.object(
                main_app, "choose_group_from_list", return_value=group.id
            ):
                with patch.object(main_app, "run_stock_menu") as run_stock_menu:
                    main_app.run_group_menu(console, container)

    run_stock_menu.assert_called_once()


def test_run_group_menu_edit_flow_invokes_update_group():
    """Should call update flow when selecting edit in group menu."""
    console = Console(record=True, width=80)
    container = MagicMock()
    group = main_app.Group(
        id=uuid4(),
        name="Tech",
        target_percentage=0.0,
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )

    with patch.object(main_app, "_load_groups", return_value=[group]):
        with patch.object(main_app, "choose_group_menu", side_effect=["edit", "back"]):
            with patch.object(
                main_app, "choose_group_from_list", return_value=group.id
            ):
                with patch.object(main_app, "update_group_flow") as update_group_flow:
                    main_app.run_group_menu(console, container)

    update_group_flow.assert_called_once()


def test_run_group_menu_displays_current_group_after_selection():
    """Should display current group after a selection."""
    console = Console(record=True, width=80)
    container = MagicMock()
    group = Group(
        id=uuid4(),
        name="Tech Stocks",
        target_percentage=0.0,
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )

    with patch.object(main_app, "_load_groups", return_value=[group]):
        with patch.object(
            main_app, "choose_group_menu", side_effect=["select", "back"]
        ):
            with patch.object(
                main_app, "choose_group_from_list", return_value=group.id
            ):
                with patch.object(main_app, "run_stock_menu"):
                    main_app.run_group_menu(console, container)

    output = console.export_text()
    assert "Current Group" in output
    assert "Tech Stocks" in output


def test_choose_group_menu_returns_selected_action():
    """Should return the selected action from chooser."""
    chooser = MagicMock(return_value="add")

    action = choose_group_menu(chooser)

    chooser.assert_called_once()
    assert action == "add"


def test_choose_group_from_list_returns_group_id():
    """Should return the selected group id."""
    group_id = uuid4()
    groups = [
        Group(
            id=group_id,
            name="Tech",
            target_percentage=0.0,
            created_at=datetime.now(),
            updated_at=datetime.now(),
        ),
    ]
    chooser = MagicMock(return_value=group_id)

    result = choose_group_from_list(groups, chooser)

    chooser.assert_called_once()
    assert result == group_id


def test_update_group_flow_updates_group_and_reports_changes():
    """Should update group and render confirmation."""
    console = Console(record=True, width=80)
    repo = MagicMock()
    group = Group(
        id=uuid4(),
        name="Old Name",
        target_percentage=10.0,
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    updated_group = Group(
        id=group.id,
        name="New Name",
        target_percentage=20.0,
        created_at=datetime.now(),
        updated_at=datetime.now(),
    )
    repo.update.return_value = updated_group

    # Mock prompt to return name then percentage
    prompt_mock = MagicMock(side_effect=["New Name", "20.0"])

    update_group_flow(console, repo, group, prompt=prompt_mock)

    repo.update.assert_called_once_with(
        group.id, name="New Name", target_percentage=20.0
    )
    output = console.export_text()
    assert "New Name" in output
    assert "20.0%" in output
