"""Tests for RebalanceService."""

from datetime import datetime
from decimal import Decimal
from uuid import uuid4

from portfolio_manager.models import Group, Stock
from portfolio_manager.models.rebalance import GroupRebalanceAction
from portfolio_manager.services.portfolio_service import (
    PortfolioSummary,
    StockHoldingWithPrice,
)
from portfolio_manager.services.rebalance_service import (
    REASON_WITHIN_TOLERANCE,
    RebalanceService,
)


def make_group(name: str, target_percentage: float) -> Group:
    """Create a test group."""
    now = datetime.now()
    return Group(
        id=uuid4(),
        name=name,
        created_at=now,
        updated_at=now,
        target_percentage=target_percentage,
    )


def make_stock(ticker: str, group_id) -> Stock:
    """Create a test stock."""
    now = datetime.now()
    return Stock(
        id=uuid4(),
        ticker=ticker,
        group_id=group_id,
        created_at=now,
        updated_at=now,
    )


def make_holding(
    stock: Stock,
    quantity: Decimal,
    price: Decimal,
    currency: str,
    name: str = "",
    value_krw: Decimal | None = None,
) -> StockHoldingWithPrice:
    """Create a test holding with price."""
    return StockHoldingWithPrice(
        stock=stock,
        quantity=quantity,
        price=price,
        currency=currency,
        name=name,
        value_krw=value_krw,
    )


class TestGroupDifferenceCalculation:
    """Test group difference calculation."""

    def test_calculate_group_differences_single_group(self) -> None:
        """Should calculate difference for a single group."""
        group = make_group("US Stocks", target_percentage=60.0)
        stock = make_stock("AAPL", group.id)
        holding = make_holding(
            stock=stock,
            quantity=Decimal("10"),
            price=Decimal("150"),
            currency="USD",
            value_krw=Decimal("2000000"),
        )

        summary = PortfolioSummary(
            holdings=[(group, holding)],
            total_value=Decimal("2000000"),
        )

        service = RebalanceService()
        differences = service.calculate_group_differences(summary)

        assert len(differences) == 1
        diff = differences[0]
        assert diff.group == group
        assert diff.current_value == Decimal("2000000")
        assert diff.target_value == Decimal("1200000")  # 60% of 2M
        assert diff.difference == Decimal("800000")  # current - target (overweight)

    def test_calculate_group_differences_multiple_groups(self) -> None:
        """Should calculate differences for multiple groups."""
        group_us = make_group("US Stocks", target_percentage=40.0)
        group_kr = make_group("KR Stocks", target_percentage=60.0)

        stock_us = make_stock("AAPL", group_us.id)
        stock_kr = make_stock("005930", group_kr.id)

        holding_us = make_holding(
            stock=stock_us,
            quantity=Decimal("10"),
            price=Decimal("150"),
            currency="USD",
            value_krw=Decimal("2000000"),
        )
        holding_kr = make_holding(
            stock=stock_kr,
            quantity=Decimal("100"),
            price=Decimal("80000"),
            currency="KRW",
            value_krw=Decimal("8000000"),
        )

        summary = PortfolioSummary(
            holdings=[(group_us, holding_us), (group_kr, holding_kr)],
            total_value=Decimal("10000000"),
        )

        service = RebalanceService()
        differences = service.calculate_group_differences(summary)

        assert len(differences) == 2

        us_diff = next(d for d in differences if d.group.name == "US Stocks")
        kr_diff = next(d for d in differences if d.group.name == "KR Stocks")

        # US: current 2M, target 4M (40%), difference -2M (underweight)
        assert us_diff.current_value == Decimal("2000000")
        assert us_diff.target_value == Decimal("4000000")
        assert us_diff.difference == Decimal("-2000000")

        # KR: current 8M, target 6M (60%), difference +2M (overweight)
        assert kr_diff.current_value == Decimal("8000000")
        assert kr_diff.target_value == Decimal("6000000")
        assert kr_diff.difference == Decimal("2000000")

    def test_calculate_group_differences_multiple_stocks_same_group(self) -> None:
        """Should sum holdings for stocks in the same group."""
        group = make_group("US Stocks", target_percentage=50.0)

        stock1 = make_stock("AAPL", group.id)
        stock2 = make_stock("GOOGL", group.id)

        holding1 = make_holding(
            stock=stock1,
            quantity=Decimal("10"),
            price=Decimal("150"),
            currency="USD",
            value_krw=Decimal("2000000"),
        )
        holding2 = make_holding(
            stock=stock2,
            quantity=Decimal("5"),
            price=Decimal("200"),
            currency="USD",
            value_krw=Decimal("1500000"),
        )

        summary = PortfolioSummary(
            holdings=[(group, holding1), (group, holding2)],
            total_value=Decimal("3500000"),
        )

        service = RebalanceService()
        differences = service.calculate_group_differences(summary)

        assert len(differences) == 1
        diff = differences[0]
        assert diff.current_value == Decimal("3500000")  # sum of both holdings
        assert diff.target_value == Decimal("1750000")  # 50% of total
        assert diff.difference == Decimal("1750000")  # overweight


class TestGroupRebalanceLogicV2:
    """Test portfolio rebalancing logic v2 at group level."""

    def test_group_under_target_by_two_percent_is_buy(self) -> None:
        """Groups at target -2% or lower should be BUY."""
        group = make_group("Overseas Growth", target_percentage=50.0)
        stock = make_stock("AAPL", group.id)
        holding = make_holding(
            stock=stock,
            quantity=Decimal("10"),
            price=Decimal("150"),
            currency="USD",
            value_krw=Decimal("4800000"),
        )

        summary = PortfolioSummary(
            holdings=[(group, holding)],
            total_value=Decimal("10000000"),
        )

        service = RebalanceService()
        actions = service.get_group_actions_v2(summary)

        assert len(actions) == 1
        assert actions[0].group == group
        assert actions[0].action == GroupRebalanceAction.BUY

    def test_group_over_target_by_two_to_four_percent_is_no_action(self) -> None:
        """Groups at target +2%~+4% should be NO_ACTION."""
        group = make_group("Overseas Growth", target_percentage=50.0)
        stock = make_stock("AAPL", group.id)
        holding = make_holding(
            stock=stock,
            quantity=Decimal("10"),
            price=Decimal("150"),
            currency="USD",
            value_krw=Decimal("5300000"),
        )

        summary = PortfolioSummary(
            holdings=[(group, holding)],
            total_value=Decimal("10000000"),
        )

        group_metrics = {
            group.id: {
                "return_since_rebalance": Decimal("25"),
                "momentum_6m": Decimal("-0.05"),
            }
        }

        service = RebalanceService()
        actions = service.get_group_actions_v2(summary, group_metrics=group_metrics)

        assert len(actions) == 1
        assert actions[0].group == group
        assert actions[0].action == GroupRebalanceAction.NO_ACTION
        assert actions[0].reason == REASON_WITHIN_TOLERANCE

    def test_group_over_target_by_four_percent_with_metrics_is_sell_candidate(
        self,
    ) -> None:
        """Groups over target +4% with return/momentum gates should be SELL_CANDIDATE."""
        group = make_group("Overseas Growth", target_percentage=50.0)
        stock = make_stock("AAPL", group.id)
        holding = make_holding(
            stock=stock,
            quantity=Decimal("10"),
            price=Decimal("150"),
            currency="USD",
            value_krw=Decimal("5500000"),
        )

        summary = PortfolioSummary(
            holdings=[(group, holding)],
            total_value=Decimal("10000000"),
        )

        group_metrics = {
            group.id: {
                "return_since_rebalance": Decimal("25"),
                "momentum_6m": Decimal("-0.02"),
            }
        }

        service = RebalanceService()
        actions = service.get_group_actions_v2(summary, group_metrics=group_metrics)

        assert len(actions) == 1
        assert actions[0].group == group
        assert actions[0].action == GroupRebalanceAction.SELL_CANDIDATE
        assert actions[0].manual_review_required is True

    def test_dividend_group_never_sell_candidate(self) -> None:
        """Dividend groups should never return SELL_CANDIDATE."""
        group = make_group("Dividend Income", target_percentage=50.0)
        stock = make_stock("DVY", group.id)
        holding = make_holding(
            stock=stock,
            quantity=Decimal("10"),
            price=Decimal("100"),
            currency="USD",
            value_krw=Decimal("5600000"),
        )

        summary = PortfolioSummary(
            holdings=[(group, holding)],
            total_value=Decimal("10000000"),
        )

        group_metrics = {
            group.id: {
                "return_since_rebalance": Decimal("30"),
                "momentum_6m": Decimal("-0.05"),
                "asset_class": "dividend",
            }
        }

        service = RebalanceService()
        actions = service.get_group_actions_v2(summary, group_metrics=group_metrics)

        assert len(actions) == 1
        assert actions[0].group == group
        assert actions[0].action == GroupRebalanceAction.NO_ACTION

    def test_growth_group_wide_sell_band_requires_above_five_percent(self) -> None:
        """Growth groups require a wider sell band than +5% to be SELL_CANDIDATE."""
        group = make_group("Overseas Growth", target_percentage=50.0)
        stock = make_stock("QQQ", group.id)
        holding = make_holding(
            stock=stock,
            quantity=Decimal("10"),
            price=Decimal("100"),
            currency="USD",
            value_krw=Decimal("5500000"),
        )

        summary = PortfolioSummary(
            holdings=[(group, holding)],
            total_value=Decimal("10000000"),
        )

        group_metrics = {
            group.id: {
                "return_since_rebalance": Decimal("25"),
                "momentum_6m": Decimal("-0.03"),
                "asset_class": "growth",
            }
        }

        service = RebalanceService()
        actions = service.get_group_actions_v2(summary, group_metrics=group_metrics)

        assert len(actions) == 1
        assert actions[0].group == group
        assert actions[0].action == GroupRebalanceAction.NO_ACTION

    def test_zero_total_value_returns_no_action(self) -> None:
        """Groups with zero total portfolio value should return NO_ACTION."""
        group = make_group("Empty Portfolio", target_percentage=50.0)
        stock = make_stock("AAPL", group.id)
        holding = make_holding(
            stock=stock,
            quantity=Decimal("0"),
            price=Decimal("150"),
            currency="USD",
            value_krw=Decimal("0"),
        )

        summary = PortfolioSummary(
            holdings=[(group, holding)],
            total_value=Decimal("0"),
        )

        service = RebalanceService()
        actions = service.get_group_actions_v2(summary)

        assert len(actions) == 1
        assert actions[0].group == group
        assert actions[0].action == GroupRebalanceAction.NO_ACTION
        assert actions[0].delta == Decimal("0")

    def test_missing_metrics_keys_defaults_to_no_action(self) -> None:
        """Groups with incomplete metrics should not trigger SELL_CANDIDATE."""
        group = make_group("Overseas Growth", target_percentage=50.0)
        stock = make_stock("AAPL", group.id)
        holding = make_holding(
            stock=stock,
            quantity=Decimal("10"),
            price=Decimal("150"),
            currency="USD",
            value_krw=Decimal("5500000"),
        )

        summary = PortfolioSummary(
            holdings=[(group, holding)],
            total_value=Decimal("10000000"),
        )

        # Metrics missing return_since_rebalance and momentum_6m
        group_metrics = {
            group.id: {
                "asset_class": "standard",
            }
        }

        service = RebalanceService()
        actions = service.get_group_actions_v2(summary, group_metrics=group_metrics)

        assert len(actions) == 1
        assert actions[0].group == group
        # Should be NO_ACTION because SELL gates require metrics
        assert actions[0].action == GroupRebalanceAction.NO_ACTION

    def test_signal_includes_delta_value(self) -> None:
        """GroupRebalanceSignal should include the computed delta."""
        group = make_group("US Stocks", target_percentage=40.0)
        stock = make_stock("AAPL", group.id)
        holding = make_holding(
            stock=stock,
            quantity=Decimal("10"),
            price=Decimal("150"),
            currency="USD",
            value_krw=Decimal("5000000"),  # 50% of total
        )

        summary = PortfolioSummary(
            holdings=[(group, holding)],
            total_value=Decimal("10000000"),
        )

        service = RebalanceService()
        actions = service.get_group_actions_v2(summary)

        assert len(actions) == 1
        # 50% current - 40% target = +10% delta
        assert actions[0].delta == Decimal("10")
