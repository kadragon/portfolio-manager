"""Test portfolio service."""

from decimal import Decimal
from unittest.mock import Mock
from uuid import uuid4

from portfolio_manager.models import Group, Stock
from portfolio_manager.services.portfolio_service import (
    PortfolioService,
    StockHoldingWithPrice,
)


def test_get_holdings_by_group():
    """그룹별로 주식과 합산 수량을 조회한다."""
    # Given: 그룹과 주식
    group1_id = uuid4()
    group2_id = uuid4()
    stock1_id = uuid4()
    stock2_id = uuid4()

    group_repo = Mock()
    group_repo.list_all.return_value = [
        Group(
            id=group1_id,
            name="Tech",
            created_at=None,  # type: ignore[arg-type]
            updated_at=None,  # type: ignore[arg-type]
        ),
        Group(
            id=group2_id,
            name="Finance",
            created_at=None,  # type: ignore[arg-type]
            updated_at=None,  # type: ignore[arg-type]
        ),
    ]

    stock_repo = Mock()
    stock_repo.list_by_group.side_effect = lambda group_id: (
        [
            Stock(
                id=stock1_id,
                ticker="AAPL",
                group_id=group1_id,
                created_at=None,  # type: ignore[arg-type]
                updated_at=None,  # type: ignore[arg-type]
            )
        ]
        if group_id == group1_id
        else [
            Stock(
                id=stock2_id,
                ticker="JPM",
                group_id=group2_id,
                created_at=None,  # type: ignore[arg-type]
                updated_at=None,  # type: ignore[arg-type]
            )
        ]
    )

    holding_repo = Mock()
    holding_repo.get_aggregated_holdings_by_stock.return_value = {
        stock1_id: Decimal("15"),
        stock2_id: Decimal("20"),
    }

    service = PortfolioService(group_repo, stock_repo, holding_repo)

    # When: 그룹별 보유 현황을 조회
    result = service.get_holdings_by_group()

    # Then: 그룹별로 주식과 합산 수량이 반환됨
    assert len(result) == 2
    assert result[0].group.id == group1_id
    assert len(result[0].stock_holdings) == 1
    assert result[0].stock_holdings[0].stock.ticker == "AAPL"
    assert result[0].stock_holdings[0].quantity == Decimal("15")
    assert result[1].group.id == group2_id
    assert len(result[1].stock_holdings) == 1
    assert result[1].stock_holdings[0].stock.ticker == "JPM"
    assert result[1].stock_holdings[0].quantity == Decimal("20")


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
        stock=stock,
        quantity=Decimal("10"),
        price=Decimal("150.0"),
        currency="USD",
        name="Apple Inc.",
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
        (Decimal("150.0"), "USD", "Apple Inc.")
        if ticker == "AAPL"
        else (Decimal("100.0"), "USD", "Google")
    )

    exchange_rate_service = Mock()
    exchange_rate_service.get_usd_krw_rate.return_value = Decimal("1300")

    # Mocks for account/deposit (optional for this test, but needed if logic changes)
    account_repo = Mock()
    account_repo.list_all.return_value = []
    deposit_repo = Mock()

    portfolio_service = PortfolioService(
        group_repo,
        stock_repo,
        holding_repo,
        price_service,
        exchange_rate_service,
        account_repository=account_repo,
        deposit_repository=deposit_repo,
    )

    # When: Get portfolio summary with valuations
    summary = portfolio_service.get_portfolio_summary()

    # Then: Total value is sum of all valuations
    assert len(summary.holdings) == 2
    assert summary.holdings[0][1].value == Decimal("1500.0")  # 10 × 150
    assert summary.holdings[1][1].value == Decimal("500.0")  # 5 × 100
    # Note: total_value will be deprecated or aliased to total_stock_value
    # Assuming we keep total_value for backward compat or update it.
    # If we update PortfolioSummary, we should check total_stock_value if renamed.
    # For now let's assume I will add fields, not rename total_value yet or alias it.
    assert summary.total_value == Decimal("2600000.0")  # (1500 + 500) * 1300


def test_portfolio_summary_sets_value_krw_for_usd_holdings():
    """USD 보유분은 KRW 환산 금액을 함께 저장한다."""
    group_id = uuid4()
    stock_id = uuid4()

    group_repo = Mock()
    group_repo.list_all.return_value = [
        Group(
            id=group_id,
            name="Overseas",
            created_at=None,  # type: ignore[arg-type]
            updated_at=None,  # type: ignore[arg-type]
        )
    ]

    stock_repo = Mock()
    stock_repo.list_by_group.return_value = [
        Stock(
            id=stock_id,
            ticker="VYM",
            group_id=group_id,
            created_at=None,  # type: ignore[arg-type]
            updated_at=None,  # type: ignore[arg-type]
        )
    ]

    holding_repo = Mock()
    holding_repo.get_aggregated_holdings_by_stock.return_value = {
        stock_id: Decimal("10"),
    }

    price_service = Mock()
    price_service.get_stock_price.return_value = (
        Decimal("100.0"),
        "USD",
        "Vanguard High Dividend Yield ETF",
    )

    exchange_rate_service = Mock()
    exchange_rate_service.get_usd_krw_rate.return_value = Decimal("1300")

    account_repo = Mock()
    account_repo.list_all.return_value = []
    deposit_repo = Mock()

    portfolio_service = PortfolioService(
        group_repo,
        stock_repo,
        holding_repo,
        price_service,
        exchange_rate_service,
        account_repository=account_repo,
        deposit_repository=deposit_repo,
    )

    summary = portfolio_service.get_portfolio_summary()

    holding = summary.holdings[0][1]
    assert holding.value_krw == Decimal("1300000.0")


def test_portfolio_summary_calculates_return_rate():
    """총 평가금액과 투자원금을 비교하여 수익률을 계산한다."""
    from datetime import datetime

    group_id = uuid4()
    stock_id = uuid4()
    now = datetime.now()

    group_repo = Mock()
    group_repo.list_all.return_value = [
        Group(id=group_id, name="G", created_at=now, updated_at=now)
    ]
    stock_repo = Mock()
    stock_repo.list_by_group.return_value = [
        Stock(
            id=stock_id, ticker="S", group_id=group_id, created_at=now, updated_at=now
        )
    ]
    holding_repo = Mock()
    holding_repo.get_aggregated_holdings_by_stock.return_value = {
        stock_id: Decimal("10")
    }
    price_service = Mock()
    price_service.get_stock_price.return_value = (Decimal("100000"), "KRW", "Samsung")

    account_id = uuid4()
    account_repo = Mock()
    account_repo.list_all.return_value = [
        Mock(id=account_id, cash_balance=Decimal("100000"))
    ]
    deposit_repo = Mock()
    deposit_repo.get_total_by_account.return_value = Decimal("1000000")

    portfolio_service = PortfolioService(
        group_repo,
        stock_repo,
        holding_repo,
        price_service,
        Mock(),
        account_repository=account_repo,
        deposit_repository=deposit_repo,
    )

    summary = portfolio_service.get_portfolio_summary()

    assert summary.total_value == Decimal("1000000")  # Stock value
    assert summary.total_cash_balance == Decimal("100000")
    assert summary.total_assets == Decimal("1100000")
    assert summary.total_invested == Decimal("1000000")
    assert summary.return_rate == Decimal("10.0")  # 10%
