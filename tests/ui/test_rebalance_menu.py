"""Tests for rebalance menu and rendering."""

from decimal import Decimal
from unittest.mock import MagicMock
from rich.console import Console

from portfolio_manager.cli.prompt_select import choose_main_menu


class TestMainMenuRebalanceOption:
    """Test main menu rebalance option."""

    def test_main_menu_includes_rebalance_option(self) -> None:
        """Main menu should include rebalance option."""
        chooser = MagicMock(return_value="rebalance")

        action = choose_main_menu(chooser)

        # Verify chooser was called with options including rebalance
        call_args = chooser.call_args
        options = call_args[1]["options"]
        option_values = [opt[0] for opt in options]

        assert "rebalance" in option_values
        assert action == "rebalance"


class TestRebalanceMenuUsesV2:
    """Test rebalance menu uses v2 signals."""

    def test_run_rebalance_menu_uses_group_actions_v2(self, monkeypatch) -> None:
        """Rebalance menu should call v2 group actions."""
        from portfolio_manager.cli import main as cli_main

        summary = MagicMock()

        class FakePortfolioService:
            def get_portfolio_summary(self):
                return summary

        class FakeContainer:
            price_service = object()

            def get_portfolio_service(self):
                return FakePortfolioService()

        class FakeRebalanceService:
            def __init__(self):
                self.group_actions_called = False

            def get_group_actions_v2(self, summary_arg):
                assert summary_arg is summary
                self.group_actions_called = True
                return ["signal"]

        fake_service = FakeRebalanceService()
        render_mock = MagicMock()

        monkeypatch.setattr(cli_main, "RebalanceService", lambda: fake_service)
        monkeypatch.setattr(cli_main, "render_rebalance_actions", render_mock)
        monkeypatch.setattr(cli_main, "Prompt", MagicMock())

        console = Console(record=True, width=120)

        cli_main.run_rebalance_menu(console, FakeContainer())  # type: ignore[arg-type]

        assert fake_service.group_actions_called


class TestLegacyRebalanceMethodRemoval:
    """Test legacy rebalance method removal."""

    def test_rebalance_service_has_no_v1_methods(self) -> None:
        """RebalanceService should not expose v1 recommendation methods."""
        from portfolio_manager.services.rebalance_service import RebalanceService

        service = RebalanceService()
        assert not hasattr(service, "get_sell_recommendations")
        assert not hasattr(service, "get_buy_recommendations")


class TestRebalanceGroupActionsRendering:
    """Test rebalance group action table rendering."""

    def test_render_rebalance_actions_shows_group_actions(self) -> None:
        """Should render group actions with delta and manual review flag."""
        from datetime import datetime
        from uuid import uuid4

        from portfolio_manager.cli.rebalance import render_rebalance_actions
        from portfolio_manager.models import Group
        from portfolio_manager.models.rebalance import (
            GroupRebalanceAction,
            GroupRebalanceSignal,
        )

        console = Console(record=True, width=120)

        group_one = Group(
            id=uuid4(),
            name="US Stocks",
            created_at=datetime.now(),
            updated_at=datetime.now(),
            target_percentage=60.0,
        )
        group_two = Group(
            id=uuid4(),
            name="KR Stocks",
            created_at=datetime.now(),
            updated_at=datetime.now(),
            target_percentage=40.0,
        )

        actions = [
            GroupRebalanceSignal(
                group=group_one,
                action=GroupRebalanceAction.BUY,
                delta=Decimal("-3.5"),
                manual_review_required=False,
            ),
            GroupRebalanceSignal(
                group=group_two,
                action=GroupRebalanceAction.SELL_CANDIDATE,
                delta=Decimal("6.2"),
                manual_review_required=True,
            ),
            GroupRebalanceSignal(
                group=group_two,
                action=GroupRebalanceAction.NO_ACTION,
                delta=Decimal("1.0"),
                manual_review_required=False,
                reason="Within tolerance band",
            ),
        ]

        render_rebalance_actions(console, actions)

        output = console.export_text()
        assert "Group" in output
        assert "Action" in output
        assert "Delta" in output
        assert "Manual" in output
        assert "Reason" in output
        assert "US Stocks" in output
        assert "KR Stocks" in output
        assert "BUY" in output
        assert "SELL" in output
        assert "Yes" in output
        assert "Within tolerance band" in output

    def test_render_rebalance_actions_empty_shows_balanced_message(self) -> None:
        """Empty actions should show a balanced message."""
        from portfolio_manager.cli.rebalance import render_rebalance_actions

        console = Console(record=True, width=120)

        render_rebalance_actions(console, [])

        output = console.export_text().lower()
        assert "balanced" in output or "no rebalancing" in output


class TestRebalanceMenuEmptyPortfolio:
    """Test rebalance menu empty portfolio handling."""

    def test_run_rebalance_menu_handles_missing_price_service_and_empty_portfolio(
        self,
        monkeypatch,
    ) -> None:
        """Should warn without prices and still render empty v2 actions."""
        from portfolio_manager.cli import main as cli_main

        console = Console(record=True, width=120)

        class NoPriceContainer:
            price_service = None

            def get_portfolio_service(self):
                return MagicMock()

        cli_main.run_rebalance_menu(console, NoPriceContainer())  # type: ignore[arg-type]

        output = console.export_text()
        assert "Price service not available" in output

        class EmptyPortfolioService:
            def get_portfolio_summary(self):
                summary = MagicMock()
                summary.total_value = Decimal("0")
                summary.holdings = []
                return summary

        class WithPriceContainer:
            price_service = object()

            def get_portfolio_service(self):
                return EmptyPortfolioService()

        render_mock = MagicMock()
        monkeypatch.setattr(
            cli_main, "render_rebalance_actions", render_mock, raising=False
        )
        monkeypatch.setattr(cli_main, "Prompt", MagicMock())

        console = Console(record=True, width=120)
        cli_main.run_rebalance_menu(console, WithPriceContainer())  # type: ignore[arg-type]

        render_mock.assert_called_once()
        actions = render_mock.call_args[0][1]
        assert actions == []


class TestLegacyRebalanceRenderRemoval:
    """Test legacy rebalance render helper removal."""

    def test_render_recommendations_removed(self) -> None:
        """render_rebalance_recommendations should be removed."""
        import portfolio_manager.cli.rebalance as rebalance

        assert not hasattr(rebalance, "render_rebalance_recommendations")
