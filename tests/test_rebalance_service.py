"""Tests for RebalanceService."""

from datetime import datetime
from decimal import Decimal
from uuid import uuid4

from portfolio_manager.models import Group, Stock
from portfolio_manager.models.rebalance import RebalanceAction
from portfolio_manager.services.portfolio_service import (
    PortfolioSummary,
    StockHoldingWithPrice,
)
from portfolio_manager.services.rebalance_service import RebalanceService


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


class TestSellRecommendations:
    """Test sell recommendations with overseas stock priority."""

    def test_sell_recommendations_include_quantity(self) -> None:
        """Sell recommendations should include calculated share quantity."""
        group = make_group("US Stocks", target_percentage=50.0)

        stock = make_stock("AAPL", group.id)
        holding = make_holding(
            stock=stock,
            quantity=Decimal("10"),
            price=Decimal("150"),
            currency="USD",
            value_krw=Decimal("2000000"),
        )

        # Total 2M, target 50% = 1M, need to sell 1M (half the holding)
        summary = PortfolioSummary(
            holdings=[(group, holding)],
            total_value=Decimal("2000000"),
        )

        service = RebalanceService()
        recommendations = service.get_sell_recommendations(summary)

        assert recommendations[0].quantity == Decimal("5")

    def test_sell_recommendations_overseas_first(self) -> None:
        """When selling, should recommend overseas stocks (USD) first."""
        group = make_group("Mixed Portfolio", target_percentage=50.0)

        stock_us = make_stock("AAPL", group.id)
        stock_kr = make_stock("005930", group.id)

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

        # Total 10M, target 50% = 5M, current = 10M, need to sell 5M
        summary = PortfolioSummary(
            holdings=[(group, holding_us), (group, holding_kr)],
            total_value=Decimal("10000000"),
        )

        service = RebalanceService()
        recommendations = service.get_sell_recommendations(summary)

        # Should recommend selling overseas stock first
        assert len(recommendations) > 0
        first_rec = recommendations[0]
        assert first_rec.ticker == "AAPL"
        assert first_rec.action == RebalanceAction.SELL
        assert first_rec.currency == "USD"

    def test_sell_recommendations_overseas_exhausted_then_domestic(self) -> None:
        """When overseas holdings exhausted, should recommend domestic stocks."""
        group = make_group("Mixed Portfolio", target_percentage=20.0)

        stock_us = make_stock("AAPL", group.id)
        stock_kr = make_stock("005930", group.id)

        # Small overseas position
        holding_us = make_holding(
            stock=stock_us,
            quantity=Decimal("5"),
            price=Decimal("150"),
            currency="USD",
            value_krw=Decimal("1000000"),  # 1M
        )
        # Large domestic position
        holding_kr = make_holding(
            stock=stock_kr,
            quantity=Decimal("100"),
            price=Decimal("90000"),
            currency="KRW",
            value_krw=Decimal("9000000"),  # 9M
        )

        # Total 10M, target 20% = 2M, current = 10M, need to sell 8M
        # Overseas only has 1M, so need to sell 7M from domestic
        summary = PortfolioSummary(
            holdings=[(group, holding_us), (group, holding_kr)],
            total_value=Decimal("10000000"),
        )

        service = RebalanceService()
        recommendations = service.get_sell_recommendations(summary)

        # Should have recommendations for both, overseas first
        assert len(recommendations) >= 2

        # First should be overseas
        assert recommendations[0].ticker == "AAPL"
        assert recommendations[0].currency == "USD"

        # Second should be domestic
        assert recommendations[1].ticker == "005930"
        assert recommendations[1].currency == "KRW"

    def test_sell_recommendations_only_for_overweight_groups(self) -> None:
        """Should only generate sell recommendations for overweight groups."""
        group_overweight = make_group("Overweight", target_percentage=30.0)
        group_underweight = make_group("Underweight", target_percentage=70.0)

        stock_over = make_stock("AAPL", group_overweight.id)
        stock_under = make_stock("005930", group_underweight.id)

        holding_over = make_holding(
            stock=stock_over,
            quantity=Decimal("10"),
            price=Decimal("150"),
            currency="USD",
            value_krw=Decimal("5000000"),  # 5M (overweight: target 3M)
        )
        holding_under = make_holding(
            stock=stock_under,
            quantity=Decimal("50"),
            price=Decimal("100000"),
            currency="KRW",
            value_krw=Decimal("5000000"),  # 5M (underweight: target 7M)
        )

        summary = PortfolioSummary(
            holdings=[
                (group_overweight, holding_over),
                (group_underweight, holding_under),
            ],
            total_value=Decimal("10000000"),
        )

        service = RebalanceService()
        recommendations = service.get_sell_recommendations(summary)

        # Should only have sell recommendations for overweight group
        for rec in recommendations:
            assert rec.action == RebalanceAction.SELL
            # Should only recommend selling AAPL (from overweight group)
            assert rec.ticker == "AAPL"


class TestBuyRecommendations:
    """Test buy recommendations with domestic stock priority."""

    def test_buy_recommendations_domestic_first(self) -> None:
        """When buying, should recommend domestic stocks (KRW) first."""
        group = make_group("Mixed Portfolio", target_percentage=100.0)

        stock_us = make_stock("AAPL", group.id)
        stock_kr = make_stock("005930", group.id)

        holding_us = make_holding(
            stock=stock_us,
            quantity=Decimal("5"),
            price=Decimal("150"),
            currency="USD",
            value_krw=Decimal("1000000"),  # 1M
        )
        holding_kr = make_holding(
            stock=stock_kr,
            quantity=Decimal("50"),
            price=Decimal("80000"),
            currency="KRW",
            value_krw=Decimal("4000000"),  # 4M
        )

        # Total 5M, target 100% = 10M (with total_value=10M), need to buy 5M
        summary = PortfolioSummary(
            holdings=[(group, holding_us), (group, holding_kr)],
            total_value=Decimal("10000000"),
        )

        service = RebalanceService()
        recommendations = service.get_buy_recommendations(summary)

        # Should recommend buying domestic stock first
        assert len(recommendations) > 0
        first_rec = recommendations[0]
        assert first_rec.ticker == "005930"
        assert first_rec.action == RebalanceAction.BUY
        assert first_rec.currency == "KRW"

    def test_buy_recommendations_only_for_underweight_groups(self) -> None:
        """Should only generate buy recommendations for underweight groups."""
        group_overweight = make_group("Overweight", target_percentage=30.0)
        group_underweight = make_group("Underweight", target_percentage=70.0)

        stock_over = make_stock("AAPL", group_overweight.id)
        stock_under = make_stock("005930", group_underweight.id)

        holding_over = make_holding(
            stock=stock_over,
            quantity=Decimal("10"),
            price=Decimal("150"),
            currency="USD",
            value_krw=Decimal("5000000"),  # 5M (overweight: target 3M)
        )
        holding_under = make_holding(
            stock=stock_under,
            quantity=Decimal("50"),
            price=Decimal("100000"),
            currency="KRW",
            value_krw=Decimal("5000000"),  # 5M (underweight: target 7M)
        )

        summary = PortfolioSummary(
            holdings=[
                (group_overweight, holding_over),
                (group_underweight, holding_under),
            ],
            total_value=Decimal("10000000"),
        )

        service = RebalanceService()
        recommendations = service.get_buy_recommendations(summary)

        # Should only have buy recommendations for underweight group
        for rec in recommendations:
            assert rec.action == RebalanceAction.BUY
            # Should only recommend buying 005930 (from underweight group)
            assert rec.ticker == "005930"

    def test_buy_recommendations_with_multiple_domestic_stocks(self) -> None:
        """Should distribute buy amount across domestic stocks."""
        group = make_group("Korean Stocks", target_percentage=100.0)

        stock_kr1 = make_stock("005930", group.id)  # Samsung
        stock_kr2 = make_stock("000660", group.id)  # SK Hynix

        holding_kr1 = make_holding(
            stock=stock_kr1,
            quantity=Decimal("50"),
            price=Decimal("80000"),
            currency="KRW",
            value_krw=Decimal("4000000"),
        )
        holding_kr2 = make_holding(
            stock=stock_kr2,
            quantity=Decimal("30"),
            price=Decimal("200000"),
            currency="KRW",
            value_krw=Decimal("6000000"),
        )

        # Total 10M, target 100% = 20M, need to buy 10M
        summary = PortfolioSummary(
            holdings=[(group, holding_kr1), (group, holding_kr2)],
            total_value=Decimal("20000000"),
        )

        service = RebalanceService()
        recommendations = service.get_buy_recommendations(summary)

        # Should have buy recommendations for both stocks
        assert len(recommendations) >= 1
        # All should be domestic
        for rec in recommendations:
            assert rec.currency == "KRW"
            assert rec.action == RebalanceAction.BUY
