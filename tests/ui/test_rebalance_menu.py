"""Tests for rebalance menu and rendering."""

from decimal import Decimal
from unittest.mock import MagicMock

from rich.console import Console

from portfolio_manager.cli.prompt_select import choose_main_menu
from portfolio_manager.models.rebalance import RebalanceAction, RebalanceRecommendation


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


class TestRebalanceRecommendationsRendering:
    """Test rebalance recommendations table rendering."""

    def test_render_rebalance_recommendations_shows_sell_section(self) -> None:
        """Should render sell recommendations with overseas priority."""
        from portfolio_manager.cli.rebalance import render_rebalance_recommendations

        console = Console(record=True, width=120)

        sell_recommendations = [
            RebalanceRecommendation(
                ticker="AAPL",
                action=RebalanceAction.SELL,
                amount=Decimal("2000000"),
                priority=1,
                currency="USD",
                group_name="US Stocks",
            ),
        ]
        buy_recommendations = []

        render_rebalance_recommendations(
            console, sell_recommendations, buy_recommendations
        )

        output = console.export_text()
        assert "Sell" in output or "SELL" in output
        assert "AAPL" in output

    def test_render_rebalance_recommendations_shows_buy_section(self) -> None:
        """Should render buy recommendations with domestic priority."""
        from portfolio_manager.cli.rebalance import render_rebalance_recommendations

        console = Console(record=True, width=120)

        sell_recommendations = []
        buy_recommendations = [
            RebalanceRecommendation(
                ticker="005930",
                action=RebalanceAction.BUY,
                amount=Decimal("3000000"),
                priority=1,
                currency="KRW",
                group_name="KR Stocks",
            ),
        ]

        render_rebalance_recommendations(
            console, sell_recommendations, buy_recommendations
        )

        output = console.export_text()
        assert "Buy" in output or "BUY" in output
        assert "005930" in output

    def test_render_rebalance_recommendations_shows_amounts(self) -> None:
        """Should render amounts with currency symbols."""
        from portfolio_manager.cli.rebalance import render_rebalance_recommendations

        console = Console(record=True, width=120)

        sell_recommendations = [
            RebalanceRecommendation(
                ticker="AAPL",
                action=RebalanceAction.SELL,
                amount=Decimal("2000000"),
                priority=1,
                currency="USD",
                group_name="US Stocks",
            ),
        ]
        buy_recommendations = [
            RebalanceRecommendation(
                ticker="005930",
                action=RebalanceAction.BUY,
                amount=Decimal("3000000"),
                priority=1,
                currency="KRW",
                group_name="KR Stocks",
            ),
        ]

        render_rebalance_recommendations(
            console, sell_recommendations, buy_recommendations
        )

        output = console.export_text()
        # Should show amounts
        assert "2,000,000" in output or "2000000" in output
        assert "3,000,000" in output or "3000000" in output

    def test_render_rebalance_recommendations_shows_quantities(self) -> None:
        """Should render share quantities."""
        from portfolio_manager.cli.rebalance import render_rebalance_recommendations

        console = Console(record=True, width=120)

        sell_recommendations = [
            RebalanceRecommendation(
                ticker="AAPL",
                action=RebalanceAction.SELL,
                amount=Decimal("2000000"),
                priority=1,
                currency="USD",
                quantity=Decimal("17"),
                group_name="US Stocks",
            ),
        ]
        buy_recommendations = [
            RebalanceRecommendation(
                ticker="005930",
                action=RebalanceAction.BUY,
                amount=Decimal("3000000"),
                priority=1,
                currency="KRW",
                quantity=Decimal("23"),
                group_name="KR Stocks",
            ),
        ]

        render_rebalance_recommendations(
            console, sell_recommendations, buy_recommendations
        )

        output = console.export_text()
        assert "Quantity" in output
        assert "17" in output
        assert "23" in output

    def test_render_rebalance_recommendations_shows_stock_names(self) -> None:
        """Should render stock names next to tickers."""
        from portfolio_manager.cli.rebalance import render_rebalance_recommendations

        console = Console(record=True, width=120)

        sell_recommendations = [
            RebalanceRecommendation(
                ticker="AAPL",
                action=RebalanceAction.SELL,
                amount=Decimal("2000000"),
                priority=1,
                currency="USD",
                stock_name="Apple Inc.",
                group_name="US Stocks",
            ),
        ]
        buy_recommendations = [
            RebalanceRecommendation(
                ticker="005930",
                action=RebalanceAction.BUY,
                amount=Decimal("3000000"),
                priority=1,
                currency="KRW",
                stock_name="Samsung Electronics",
                group_name="KR Stocks",
            ),
        ]

        render_rebalance_recommendations(
            console, sell_recommendations, buy_recommendations
        )

        output = console.export_text()
        assert "Name" in output
        assert "Apple Inc." in output
        assert "Samsung Electronics" in output

    def test_render_rebalance_recommendations_hides_priority(self) -> None:
        """Should not render priority column."""
        from portfolio_manager.cli.rebalance import render_rebalance_recommendations

        console = Console(record=True, width=120)

        sell_recommendations = [
            RebalanceRecommendation(
                ticker="AAPL",
                action=RebalanceAction.SELL,
                amount=Decimal("2000000"),
                priority=1,
                currency="USD",
                stock_name="Apple Inc.",
                group_name="US Stocks",
            ),
        ]
        buy_recommendations = [
            RebalanceRecommendation(
                ticker="005930",
                action=RebalanceAction.BUY,
                amount=Decimal("3000000"),
                priority=1,
                currency="KRW",
                stock_name="Samsung Electronics",
                group_name="KR Stocks",
            ),
        ]

        render_rebalance_recommendations(
            console, sell_recommendations, buy_recommendations
        )

        output = console.export_text()
        assert "Priority" not in output

    def test_render_rebalance_recommendations_strips_etf_suffix(self) -> None:
        """Should strip ETF suffix from stock names."""
        from portfolio_manager.cli.rebalance import render_rebalance_recommendations

        console = Console(record=True, width=120)

        sell_recommendations = [
            RebalanceRecommendation(
                ticker="069500",
                action=RebalanceAction.SELL,
                amount=Decimal("2000000"),
                priority=1,
                currency="KRW",
                stock_name="KODEX 200 증권상장지수투자신탁(주식)",
                group_name="KR ETF",
            ),
        ]

        render_rebalance_recommendations(console, sell_recommendations, [])

        output = console.export_text()
        assert "증권상장지수투자신탁(주식)" not in output
        assert "KODEX 200" in output

    def test_render_rebalance_recommendations_empty_shows_balanced_message(
        self,
    ) -> None:
        """Should show message when portfolio is balanced."""
        from portfolio_manager.cli.rebalance import render_rebalance_recommendations

        console = Console(record=True, width=120)

        render_rebalance_recommendations(console, [], [])

        output = console.export_text()
        assert "balanced" in output.lower() or "no" in output.lower()
