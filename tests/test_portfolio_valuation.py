"""Test portfolio valuation."""

from decimal import Decimal
from unittest.mock import Mock
from uuid import uuid4

from portfolio_manager.models import Group, Stock
from portfolio_manager.services.portfolio_service import (
    PortfolioService,
    StockHoldingWithPrice,
)


def test_calculate_valuation_with_prices():
    """평가금액(수량 × 현재가)을 계산한다."""
    # Given: Stock with quantity and price
    stock = Stock(
        id=uuid4(),
        ticker="AAPL",
        group_id=uuid4(),
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )

    holding = StockHoldingWithPrice(
        stock=stock, quantity=Decimal("10"), price=Decimal("150.0"), currency="USD"
    )

    # When: Calculate valuation
    value = holding.value

    # Then: Value is quantity × price
    assert value == Decimal("1500.0")


def test_portfolio_summary_calculates_total_value():
    """총 평가금액을 계산하여 표시한다."""
    # Given: Mock repositories and price service
    group_id = uuid4()
    stock1_id = uuid4()
    stock2_id = uuid4()

    group_repo = Mock()
    group_repo.list_all.return_value = [
        Group(
            id=group_id,
            name="Tech",
            created_at=None,  # type: ignore[arg-type]
            updated_at=None,  # type: ignore[arg-type]
        )
    ]

    stock_repo = Mock()
    stock_repo.list_by_group.return_value = [
        Stock(
            id=stock1_id,
            ticker="AAPL",
            group_id=group_id,
            created_at=None,  # type: ignore[arg-type]
            updated_at=None,  # type: ignore[arg-type]
        ),
        Stock(
            id=stock2_id,
            ticker="GOOGL",
            group_id=group_id,
            created_at=None,  # type: ignore[arg-type]
            updated_at=None,  # type: ignore[arg-type]
        ),
    ]

    holding_repo = Mock()
    holding_repo.get_aggregated_holdings_by_stock.return_value = {
        stock1_id: Decimal("10"),
        stock2_id: Decimal("5"),
    }

    price_service = Mock()
    price_service.get_stock_price.side_effect = lambda ticker: (
        (Decimal("150.0"), "USD") if ticker == "AAPL" else (Decimal("100.0"), "USD")
    )

    portfolio_service = PortfolioService(
        group_repo, stock_repo, holding_repo, price_service
    )

    # When: Get portfolio summary with valuations
    summary = portfolio_service.get_portfolio_summary()

    # Then: Total value is sum of all valuations
    assert len(summary.holdings) == 2
    assert summary.holdings[0][1].value == Decimal("1500.0")  # 10 × 150
    assert summary.holdings[1][1].value == Decimal("500.0")  # 5 × 100
    assert summary.total_value == Decimal("2000.0")  # 1500 + 500
