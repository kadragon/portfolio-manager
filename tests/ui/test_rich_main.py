"""Tests for Rich-based main menu and navigation."""

from decimal import Decimal
from unittest.mock import MagicMock, patch

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
                main_app, "choose_main_menu", side_effect=["groups", "quit"]
            ):
                with patch.object(main_app, "run_group_menu"):
                    with patch.object(main_app.time, "time", return_value=1000.0):
                        main_app.main()

    assert portfolio_service.get_portfolio_summary.call_count == 1


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
