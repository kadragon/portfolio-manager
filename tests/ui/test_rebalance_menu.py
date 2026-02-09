"""Tests for rebalance menu and rendering."""

from decimal import Decimal
from unittest.mock import MagicMock, patch

import pytest
from rich.console import Console

from portfolio_manager.cli.prompt_select import (
    choose_main_menu,
    choose_rebalance_action,
)
from portfolio_manager.models.rebalance import RebalanceAction, RebalanceRecommendation
from portfolio_manager.services.rebalance_execution_service import (
    OrderExecutionResult,
    OrderIntent,
    RebalanceExecutionResult,
)


@pytest.fixture
def sample_sell_recommendations() -> list[RebalanceRecommendation]:
    return [
        RebalanceRecommendation(
            ticker="AAPL",
            action=RebalanceAction.SELL,
            amount=Decimal("2000000"),
            priority=1,
            currency="USD",
            group_name="US Stocks",
        ),
    ]


@pytest.fixture
def sample_buy_recommendations() -> list[RebalanceRecommendation]:
    return [
        RebalanceRecommendation(
            ticker="005930",
            action=RebalanceAction.BUY,
            amount=Decimal("3000000"),
            priority=1,
            currency="KRW",
            group_name="KR Stocks",
        ),
    ]


class TestMainMenuRebalanceOption:
    """Test main menu rebalance option."""

    def test_rebalance_menu_shows_preview_and_execute_options(self) -> None:
        """Rebalance menu should show Preview only and Execute orders choices."""
        chooser = MagicMock(return_value="preview")

        action = choose_rebalance_action(chooser)

        call_args = chooser.call_args
        options = call_args[1]["options"]
        option_values = [opt[0] for opt in options]
        option_labels = [opt[1] for opt in options]

        assert "preview" in option_values
        assert "execute" in option_values
        assert any("Preview" in label for label in option_labels)
        assert any("Execute" in label for label in option_labels)
        assert action == "preview"

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


class TestRebalanceExecutionFlow:
    """Test rebalance execution CLI flow."""

    def test_execute_cancelled_at_confirmation_does_not_place_orders(self) -> None:
        """When user selects Execute but cancels confirmation, no orders are placed."""
        from portfolio_manager.cli import main as main_module
        from portfolio_manager.cli.main import run_rebalance_menu

        console = Console(record=True, width=120)
        container = MagicMock()
        container.price_service = MagicMock()
        container.get_portfolio_service.return_value.get_portfolio_summary.return_value = MagicMock()

        sell_recs = [
            RebalanceRecommendation(
                ticker="AAPL",
                action=RebalanceAction.SELL,
                amount=Decimal("2000000"),
                priority=1,
                currency="USD",
                quantity=Decimal("10"),
                group_name="US Stocks",
            ),
        ]

        with patch.object(main_module, "RebalanceService") as MockRS:
            MockRS.return_value.get_sell_recommendations.return_value = sell_recs
            MockRS.return_value.get_buy_recommendations.return_value = []
            with patch.object(main_module, "Prompt"):
                with patch.object(
                    main_module, "choose_rebalance_action", return_value="execute"
                ):
                    with patch.object(main_module, "Confirm") as MockConfirm:
                        MockConfirm.ask.return_value = False
                        with patch.object(
                            main_module, "RebalanceExecutionService"
                        ) as MockExecService:
                            run_rebalance_menu(console, container)

                            MockExecService.return_value.execute_rebalance_orders.assert_not_called()

    def test_execute_confirmed_renders_result_summary(self) -> None:
        """When user confirms execution, result table with success/failed/skipped counts is shown."""
        from portfolio_manager.cli import main as main_module
        from portfolio_manager.cli.main import run_rebalance_menu

        console = Console(record=True, width=120)
        container = MagicMock()
        container.price_service = MagicMock()
        container.get_portfolio_service.return_value.get_portfolio_summary.return_value = MagicMock()

        sell_recs = [
            RebalanceRecommendation(
                ticker="AAPL",
                action=RebalanceAction.SELL,
                amount=Decimal("2000000"),
                priority=1,
                currency="USD",
                quantity=Decimal("10"),
                group_name="US Stocks",
            ),
        ]
        buy_recs = [
            RebalanceRecommendation(
                ticker="005930",
                action=RebalanceAction.BUY,
                amount=Decimal("3000000"),
                priority=1,
                currency="KRW",
                quantity=Decimal("5"),
                group_name="KR Stocks",
            ),
        ]

        intent_sell = OrderIntent(
            ticker="AAPL", side="sell", quantity=10, currency="USD", exchange="NASD"
        )
        intent_buy = OrderIntent(
            ticker="005930", side="buy", quantity=5, currency="KRW"
        )
        skipped_intent = OrderIntent(
            ticker="MSFT", side="buy", quantity=0, currency="USD", exchange="NASD"
        )

        exec_result = RebalanceExecutionResult(
            intents=[intent_sell, intent_buy],
            skipped=[skipped_intent],
            executions=[
                OrderExecutionResult(
                    intent=intent_sell, status="success", message="ok"
                ),
                OrderExecutionResult(
                    intent=intent_buy, status="failed", message="insufficient funds"
                ),
            ],
        )

        with patch.object(main_module, "RebalanceService") as MockRS:
            MockRS.return_value.get_sell_recommendations.return_value = sell_recs
            MockRS.return_value.get_buy_recommendations.return_value = buy_recs
            with patch.object(main_module, "Prompt"):
                with patch.object(
                    main_module, "choose_rebalance_action", return_value="execute"
                ):
                    with patch.object(main_module, "Confirm") as MockConfirm:
                        MockConfirm.ask.return_value = True
                        with patch.object(
                            main_module, "RebalanceExecutionService"
                        ) as MockExecService:
                            MockExecService.return_value.execute_rebalance_orders.return_value = exec_result
                            run_rebalance_menu(console, container)

        output = console.export_text()
        # Should display counts for success, failed, and skipped
        assert "1" in output  # 1 success
        assert "failed" in output.lower() or "Failed" in output
        assert "skipped" in output.lower() or "Skipped" in output

    def test_failed_orders_show_detail_info(self) -> None:
        """Failed orders should display ticker, action, msg_cd, and msg1."""
        from portfolio_manager.cli import main as main_module
        from portfolio_manager.cli.main import run_rebalance_menu

        console = Console(record=True, width=120)
        container = MagicMock()
        container.price_service = MagicMock()
        container.get_portfolio_service.return_value.get_portfolio_summary.return_value = MagicMock()

        sell_recs = [
            RebalanceRecommendation(
                ticker="AAPL",
                action=RebalanceAction.SELL,
                amount=Decimal("2000000"),
                priority=1,
                currency="USD",
                quantity=Decimal("10"),
                group_name="US Stocks",
            ),
        ]

        intent_fail = OrderIntent(
            ticker="AAPL", side="sell", quantity=10, currency="USD", exchange="NASD"
        )

        exec_result = RebalanceExecutionResult(
            intents=[intent_fail],
            skipped=[],
            executions=[
                OrderExecutionResult(
                    intent=intent_fail,
                    status="failed",
                    message="주문 실패",
                    raw_response={"msg_cd": "APBK1234", "msg1": "잔고부족"},
                ),
            ],
        )

        with patch.object(main_module, "RebalanceService") as MockRS:
            MockRS.return_value.get_sell_recommendations.return_value = sell_recs
            MockRS.return_value.get_buy_recommendations.return_value = []
            with patch.object(main_module, "Prompt"):
                with patch.object(
                    main_module, "choose_rebalance_action", return_value="execute"
                ):
                    with patch.object(main_module, "Confirm") as MockConfirm:
                        MockConfirm.ask.return_value = True
                        with patch.object(
                            main_module, "RebalanceExecutionService"
                        ) as MockExecService:
                            MockExecService.return_value.execute_rebalance_orders.return_value = exec_result
                            run_rebalance_menu(console, container)

        output = console.export_text()
        assert "AAPL" in output
        assert "sell" in output.lower()
        assert "APBK1234" in output
        assert "잔고부족" in output


class TestRebalanceRecommendationsRendering:
    """Test rebalance recommendations table rendering."""

    def test_render_rebalance_recommendations_shows_sell_section(
        self,
        sample_sell_recommendations: list[RebalanceRecommendation],
    ) -> None:
        """Should render sell recommendations with overseas priority."""
        from portfolio_manager.cli.rebalance import render_rebalance_recommendations

        console = Console(record=True, width=120)

        render_rebalance_recommendations(console, sample_sell_recommendations, [])

        output = console.export_text()
        assert "Sell" in output or "SELL" in output
        assert "AAPL" in output

    def test_render_rebalance_recommendations_shows_buy_section(
        self,
        sample_buy_recommendations: list[RebalanceRecommendation],
    ) -> None:
        """Should render buy recommendations with domestic priority."""
        from portfolio_manager.cli.rebalance import render_rebalance_recommendations

        console = Console(record=True, width=120)

        render_rebalance_recommendations(console, [], sample_buy_recommendations)

        output = console.export_text()
        assert "Buy" in output or "BUY" in output
        assert "005930" in output

    def test_render_rebalance_recommendations_shows_amounts(
        self,
        sample_sell_recommendations: list[RebalanceRecommendation],
        sample_buy_recommendations: list[RebalanceRecommendation],
    ) -> None:
        """Should render amounts with currency symbols."""
        from portfolio_manager.cli.rebalance import render_rebalance_recommendations

        console = Console(record=True, width=120)

        render_rebalance_recommendations(
            console, sample_sell_recommendations, sample_buy_recommendations
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

    def test_render_rebalance_recommendations_shows_account_columns(
        self,
        sample_sell_recommendations: list[RebalanceRecommendation],
        sample_buy_recommendations: list[RebalanceRecommendation],
    ) -> None:
        """Should render sell and buy account columns with same account."""
        from portfolio_manager.cli.rebalance import render_rebalance_recommendations

        console = Console(record=True, width=120)

        render_rebalance_recommendations(
            console, sample_sell_recommendations, sample_buy_recommendations
        )

        output = console.export_text()
        assert "Sell Account" in output
        assert "Buy Account" in output
        assert "Same Account" in output

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
