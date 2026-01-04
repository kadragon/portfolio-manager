"""Tests for RebalanceRecommendation model."""

from decimal import Decimal

from portfolio_manager.models.rebalance import RebalanceRecommendation, RebalanceAction


class TestRebalanceRecommendation:
    """Test RebalanceRecommendation dataclass."""

    def test_recommendation_has_required_fields(self) -> None:
        """RebalanceRecommendation should have ticker, action, amount, and priority."""
        recommendation = RebalanceRecommendation(
            ticker="AAPL",
            action=RebalanceAction.SELL,
            amount=Decimal("1000"),
            priority=1,
        )

        assert recommendation.ticker == "AAPL"
        assert recommendation.action == RebalanceAction.SELL
        assert recommendation.amount == Decimal("1000")
        assert recommendation.priority == 1

    def test_recommendation_buy_action(self) -> None:
        """RebalanceRecommendation should support BUY action."""
        recommendation = RebalanceRecommendation(
            ticker="005930",
            action=RebalanceAction.BUY,
            amount=Decimal("500000"),
            priority=2,
        )

        assert recommendation.action == RebalanceAction.BUY

    def test_recommendation_optional_currency_field(self) -> None:
        """RebalanceRecommendation should have optional currency field."""
        recommendation = RebalanceRecommendation(
            ticker="AAPL",
            action=RebalanceAction.SELL,
            amount=Decimal("1000"),
            priority=1,
            currency="USD",
        )

        assert recommendation.currency == "USD"

    def test_recommendation_optional_quantity_field(self) -> None:
        """RebalanceRecommendation should have optional quantity field for shares."""
        recommendation = RebalanceRecommendation(
            ticker="AAPL",
            action=RebalanceAction.SELL,
            amount=Decimal("1000"),
            priority=1,
            quantity=Decimal("5"),
        )

        assert recommendation.quantity == Decimal("5")

    def test_recommendation_optional_group_name_field(self) -> None:
        """RebalanceRecommendation should have optional group_name field."""
        recommendation = RebalanceRecommendation(
            ticker="AAPL",
            action=RebalanceAction.SELL,
            amount=Decimal("1000"),
            priority=1,
            group_name="US Stocks",
        )

        assert recommendation.group_name == "US Stocks"
