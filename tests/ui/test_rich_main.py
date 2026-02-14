"""Tests for Rich-based main menu and navigation."""

from datetime import datetime
from decimal import Decimal
from unittest.mock import MagicMock, patch
from uuid import uuid4

import pytest
from rich.console import Console

from portfolio_manager.cli.app import (
    handle_main_menu_key,
    render_main_menu,
    select_main_menu_option,
)
from portfolio_manager.cli.menu import render_menu_options
from portfolio_manager.cli.groups import select_group_menu_option
from portfolio_manager.cli import main as main_app
from portfolio_manager.cli.prompt_select import choose_main_menu
from portfolio_manager.services.portfolio_service import PortfolioSummary


def test_rich_main_menu_renders_title():
    """Should render the main menu title using Rich."""
    console = Console(record=True, width=80)

    render_main_menu(console)

    output = console.export_text()
    assert "Portfolio Manager" in output


def test_selecting_groups_option_returns_groups_action():
    """Should route to group management when selecting groups option."""
    action = select_main_menu_option("g")

    assert action == "groups"


def test_main_menu_quit_exits():
    """Should exit the main loop when selecting quit."""
    with patch("portfolio_manager.cli.main.ServiceContainer") as MockContainer:
        container = MockContainer.return_value
        container.get_portfolio_service.return_value = MagicMock()

        with patch.object(main_app, "render_main_menu") as render_menu:
            with patch.object(main_app, "choose_main_menu", return_value="quit"):
                with patch.object(main_app, "run_group_menu") as run_group_menu:
                    with patch.object(main_app, "run_account_menu") as run_account_menu:
                        main_app.main()

    render_menu.assert_called()
    run_group_menu.assert_not_called()
    run_account_menu.assert_not_called()


def test_main_menu_renders_dashboard_and_quits():
    """Should render dashboard before quitting."""
    summary = PortfolioSummary(holdings=[], total_value=Decimal("0"))

    with patch("portfolio_manager.cli.main.ServiceContainer") as MockContainer:
        container = MockContainer.return_value
        container.get_portfolio_service.return_value = MagicMock(
            get_portfolio_summary=MagicMock(return_value=summary)
        )
        container.price_service = object()
        container.setup = MagicMock()

        with patch.object(main_app, "render_dashboard") as render_dashboard:
            with patch.object(main_app, "choose_main_menu", return_value="quit"):
                main_app.main()

    render_dashboard.assert_called()
    assert render_dashboard.call_args[0][1] is summary


def test_main_menu_uses_fast_summary_without_change_rates():
    """Should skip change-rate lookup for main dashboard render."""
    summary = PortfolioSummary(holdings=[], total_value=Decimal("0"))

    with patch("portfolio_manager.cli.main.ServiceContainer") as MockContainer:
        container = MockContainer.return_value
        portfolio_service = MagicMock()
        portfolio_service.get_portfolio_summary.return_value = summary
        container.get_portfolio_service.return_value = portfolio_service
        container.price_service = object()
        container.setup = MagicMock()

        with patch.object(main_app, "render_dashboard"):
            with patch.object(main_app, "choose_main_menu", return_value="quit"):
                main_app.main()

    portfolio_service.get_portfolio_summary.assert_called_once_with(
        include_change_rates=False
    )


def test_main_menu_uses_summary_cache_within_ttl():
    """Should reuse cached summary within TTL window."""
    summary = PortfolioSummary(holdings=[], total_value=Decimal("0"))

    with patch("portfolio_manager.cli.main.ServiceContainer") as MockContainer:
        container = MockContainer.return_value
        portfolio_service = MagicMock()
        portfolio_service.get_portfolio_summary.return_value = summary
        container.get_portfolio_service.return_value = portfolio_service
        container.price_service = object()
        container.setup = MagicMock()

        with patch.object(main_app, "render_dashboard"):
            with patch.object(
                main_app, "choose_main_menu", side_effect=["unknown", "quit"]
            ):
                with patch.object(main_app.time, "time", return_value=1000.0):
                    main_app.main()

    assert portfolio_service.get_portfolio_summary.call_count == 1


def test_main_menu_invalidates_summary_cache_after_group_menu():
    """Should invalidate cached summary after group menu mutations."""
    summary = PortfolioSummary(holdings=[], total_value=Decimal("0"))

    with patch("portfolio_manager.cli.main.ServiceContainer") as MockContainer:
        container = MockContainer.return_value
        portfolio_service = MagicMock()
        portfolio_service.get_portfolio_summary.return_value = summary
        container.get_portfolio_service.return_value = portfolio_service
        container.price_service = object()
        container.setup = MagicMock()

        with patch.object(main_app, "render_dashboard"):
            with patch.object(
                main_app, "choose_main_menu", side_effect=["groups", "quit"]
            ):
                with patch.object(main_app, "run_group_menu"):
                    with patch.object(main_app.time, "time", return_value=1000.0):
                        main_app.main()

    assert portfolio_service.get_portfolio_summary.call_count == 2


def test_render_menu_options_prints_options_line():
    """Should print a standardized options line."""
    console = Console(record=True, width=80)

    render_menu_options(console, "g=groups | q=quit")

    output = console.export_text()
    assert "Options:" in output
    assert "g=groups" in output


def test_selecting_back_from_group_menu_returns_back_action():
    """Should return back action when selecting back in group menu."""
    action = select_group_menu_option("b")

    assert action == "back"


def test_main_menu_g_key_returns_groups_action():
    """Should return groups action when pressing 'g'."""
    action = handle_main_menu_key("g")

    assert action == "groups"


def test_choose_main_menu_returns_selected_action():
    """Should return the selected action from chooser."""
    chooser = MagicMock(return_value="groups")

    action = choose_main_menu(chooser)

    chooser.assert_called_once()
    assert action == "groups"


@pytest.mark.parametrize(
    ("status", "restored", "error", "is_config_error", "expected"),
    [
        (main_app.ProjectStatus.ACTIVE, False, None, False, True),
        (
            main_app.ProjectStatus.RESTORING,
            False,
            "Project restoration taking longer than expected.",
            False,
            True,
        ),
        (
            main_app.ProjectStatus.PAUSED,
            False,
            "Failed to request restore",
            False,
            False,
        ),
    ],
)
def test_ensure_supabase_ready_handles_main_status_paths(
    status: main_app.ProjectStatus,
    restored: bool,
    error: str | None,
    is_config_error: bool,
    expected: bool,
):
    """Should handle ACTIVE/RESTORING/PAUSED startup gate paths."""
    console = MagicMock()
    result = MagicMock()
    result.status = status
    result.restored = restored
    result.error = error
    result.is_config_error = is_config_error

    with patch.object(main_app, "check_and_restore_project", return_value=result):
        assert main_app._ensure_supabase_ready(console) is expected


def test_ensure_supabase_ready_continues_on_non_blocking_config_error():
    """Should continue startup when auto-restore is disabled by config."""
    console = MagicMock()
    result = MagicMock()
    result.status = main_app.ProjectStatus.UNKNOWN
    result.restored = False
    result.error = "SUPABASE_ACCESS_TOKEN not set (project auto-restore disabled)"
    result.is_config_error = True

    with patch.object(main_app, "check_and_restore_project", return_value=result):
        assert main_app._ensure_supabase_ready(console) is True


def test_select_group_by_index_handles_bounds_and_valid_selection():
    """Should return None for out-of-range index and group for valid index."""
    groups = [
        main_app.Group(
            id=uuid4(), name="A", created_at=datetime.now(), updated_at=datetime.now()
        ),
        main_app.Group(
            id=uuid4(), name="B", created_at=datetime.now(), updated_at=datetime.now()
        ),
    ]

    assert main_app._select_group_by_index(groups, 0) is None
    assert main_app._select_group_by_index(groups, 3) is None
    assert main_app._select_group_by_index(groups, 2) is groups[1]


def test_select_group_by_id_returns_none_when_not_found():
    """Should return None when no matching group id exists."""
    groups = [
        main_app.Group(
            id=uuid4(), name="A", created_at=datetime.now(), updated_at=datetime.now()
        )
    ]

    assert main_app._select_group_by_id(groups, uuid4()) is None


def test_main_returns_early_when_supabase_not_ready():
    """Should abort startup if Supabase is not ready."""
    with patch.object(main_app, "_ensure_supabase_ready", return_value=False):
        with patch("portfolio_manager.cli.main.ServiceContainer") as mock_container:
            main_app.main()

    mock_container.assert_not_called()


def test_main_falls_back_to_holdings_when_price_fetch_fails():
    """Should render group holdings fallback when summary fetch raises."""
    fallback = [{"group": "fallback"}]

    with patch("portfolio_manager.cli.main.ServiceContainer") as MockContainer:
        container = MockContainer.return_value
        portfolio_service = MagicMock()
        portfolio_service.get_portfolio_summary.side_effect = RuntimeError("price fail")
        portfolio_service.get_holdings_by_group.return_value = fallback
        container.get_portfolio_service.return_value = portfolio_service
        container.price_service = object()
        container.setup = MagicMock()

        with patch.object(main_app, "render_dashboard") as render_dashboard:
            with patch.object(main_app, "choose_main_menu", return_value="quit"):
                main_app.main()

    portfolio_service.get_holdings_by_group.assert_called_once_with()
    assert render_dashboard.call_args[0][1] is fallback


def test_main_uses_holdings_dashboard_when_price_service_absent():
    """Should render holdings dashboard directly when price service is unavailable."""
    holdings = [{"group": "no-price"}]

    with patch("portfolio_manager.cli.main.ServiceContainer") as MockContainer:
        container = MockContainer.return_value
        portfolio_service = MagicMock()
        portfolio_service.get_holdings_by_group.return_value = holdings
        container.get_portfolio_service.return_value = portfolio_service
        container.price_service = None
        container.setup = MagicMock()

        with patch.object(main_app, "render_dashboard") as render_dashboard:
            with patch.object(main_app, "choose_main_menu", return_value="quit"):
                main_app.main()

    portfolio_service.get_portfolio_summary.assert_not_called()
    portfolio_service.get_holdings_by_group.assert_called_once_with()
    assert render_dashboard.call_args[0][1] is holdings


@pytest.mark.parametrize(
    ("action", "menu_function_name"),
    [
        ("accounts", "run_account_menu"),
        ("deposits", "run_deposit_menu"),
        ("rebalance", "run_rebalance_menu"),
    ],
)
def test_main_invalidates_cache_after_non_group_menu_actions(
    action: str, menu_function_name: str
):
    """Accounts/deposits/rebalance actions should invalidate summary cache."""
    summary = PortfolioSummary(holdings=[], total_value=Decimal("0"))

    with patch("portfolio_manager.cli.main.ServiceContainer") as MockContainer:
        container = MockContainer.return_value
        portfolio_service = MagicMock()
        portfolio_service.get_portfolio_summary.return_value = summary
        container.get_portfolio_service.return_value = portfolio_service
        container.price_service = object()
        container.setup = MagicMock()

        with patch.object(main_app, "render_dashboard"):
            with patch.object(
                main_app, "choose_main_menu", side_effect=[action, "quit"]
            ):
                with patch.object(main_app, menu_function_name) as menu_function:
                    with patch.object(main_app.time, "time", return_value=1000.0):
                        main_app.main()

    menu_function.assert_called_once()
    assert portfolio_service.get_portfolio_summary.call_count == 2


def test_ensure_supabase_ready_prints_message_when_restored_active():
    """Restored ACTIVE status should print success message."""
    console = MagicMock()
    result = MagicMock()
    result.status = main_app.ProjectStatus.ACTIVE
    result.restored = True
    result.error = None
    result.is_config_error = False

    with patch.object(main_app, "check_and_restore_project", return_value=result):
        assert main_app._ensure_supabase_ready(console) is True

    assert "Supabase project restored and ready!" in str(console.print.call_args)
