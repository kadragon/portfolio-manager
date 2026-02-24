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

    def test_recommendation_optional_v2_metadata_fields(self) -> None:
        """RebalanceRecommendation should support v2 metadata fields."""
        recommendation = RebalanceRecommendation(
            ticker="AAPL",
            action=RebalanceAction.SELL,
            amount=Decimal("1000"),
            priority=1,
            account_name="ISA",
            sleeve_name="해외성장",
            reason="과열 슬리브 절반 감축",
            trigger_type="sleeve",
            amount_krw=Decimal("1300000"),
            amount_local=Decimal("1000"),
        )

        assert recommendation.account_name == "ISA"
        assert recommendation.sleeve_name == "해외성장"
        assert recommendation.reason == "과열 슬리브 절반 감축"
        assert recommendation.trigger_type == "sleeve"
        assert recommendation.amount_krw == Decimal("1300000")
        assert recommendation.amount_local == Decimal("1000")
