"""Tests for portfolio sorting by value."""

from datetime import datetime
from decimal import Decimal
from unittest.mock import MagicMock
from uuid import uuid4

from portfolio_manager.models.group import Group
from portfolio_manager.models.stock import Stock


def test_portfolio_summary_holdings_sorted_by_value_krw_descending():
    """PortfolioService should sort holdings by value_krw descending."""
    from portfolio_manager.services.portfolio_service import PortfolioService

    # Setup mock groups and stocks
    now = datetime.now()
    g1_id = uuid4()
    s1_id = uuid4()
    s2_id = uuid4()
    s3_id = uuid4()

    group1 = Group(
        id=g1_id, name="Group1", created_at=now, updated_at=now, target_percentage=50
    )
    stock1 = Stock(
        id=s1_id, ticker="005930", group_id=g1_id, created_at=now, updated_at=now
    )
    stock2 = Stock(
        id=s2_id, ticker="000660", group_id=g1_id, created_at=now, updated_at=now
    )
    stock3 = Stock(
        id=s3_id, ticker="035420", group_id=g1_id, created_at=now, updated_at=now
    )

    group_repo = MagicMock()
    group_repo.list_all.return_value = [group1]

    stock_repo = MagicMock()
    stock_repo.list_all.return_value = [stock1, stock2, stock3]

    holding_repo = MagicMock()
    holding_repo.get_aggregated_holdings_by_stock.return_value = {
        s1_id: Decimal("10"),  # quantity
        s2_id: Decimal("20"),
        s3_id: Decimal("5"),
    }

    account_repo = MagicMock()
    account_repo.list_all.return_value = []

    deposit_repo = MagicMock()
    deposit_repo.get_total.return_value = Decimal("0")
    deposit_repo.get_first_deposit_date.return_value = None

    # Mock price service to return different prices
    price_service = MagicMock()

    def mock_get_stock_price(ticker, preferred_exchange=None):
        prices = {
            "005930": (
                Decimal("70000"),
                "KRW",
                "삼성전자",
                None,
            ),  # 10 * 70000 = 700,000
            "000660": (
                Decimal("150000"),
                "KRW",
                "SK하이닉스",
                None,
            ),  # 20 * 150000 = 3,000,000
            "035420": (
                Decimal("300000"),
                "KRW",
                "NAVER",
                None,
            ),  # 5 * 300000 = 1,500,000
        }
        return prices.get(ticker, (Decimal("0"), "KRW", "", None))

    price_service.get_stock_price.side_effect = mock_get_stock_price
    price_service.get_stock_change_rates.return_value = None

    service = PortfolioService(
        group_repository=group_repo,
        stock_repository=stock_repo,
        holding_repository=holding_repo,
        price_service=price_service,
        account_repository=account_repo,
        deposit_repository=deposit_repo,
    )

    summary = service.get_portfolio_summary()

    # Holdings should be sorted by value_krw descending:
    # SK하이닉스 (3,000,000) > NAVER (1,500,000) > 삼성전자 (700,000)
    assert len(summary.holdings) == 3

    values = [
        holding.value_krw
        for _, holding in summary.holdings
        if holding.value_krw is not None
    ]
    assert len(values) == len(summary.holdings)
    assert values == sorted(values, reverse=True), (
        f"Holdings not sorted by value_krw descending: {values}"
    )

    # Verify the order
    assert summary.holdings[0][1].name == "SK하이닉스"
    assert summary.holdings[1][1].name == "NAVER"
    assert summary.holdings[2][1].name == "삼성전자"


def test_group_summary_table_sorted_by_total_value_descending():
    """Group summary table should be sorted by total value descending."""
    from rich.console import Console

    from portfolio_manager.cli.app import render_dashboard
    from portfolio_manager.services.portfolio_service import (
        PortfolioSummary,
        StockHoldingWithPrice,
    )

    # Setup groups with different total values
    now = datetime.now()
    g1_id = uuid4()
    g2_id = uuid4()
    g3_id = uuid4()

    group1 = Group(
        id=g1_id,
        name="SmallGroup",
        created_at=now,
        updated_at=now,
        target_percentage=30,
    )
    group2 = Group(
        id=g2_id,
        name="BigGroup",
        created_at=now,
        updated_at=now,
        target_percentage=40,
    )
    group3 = Group(
        id=g3_id,
        name="MediumGroup",
        created_at=now,
        updated_at=now,
        target_percentage=30,
    )

    stock1 = Stock(
        id=uuid4(), ticker="005930", group_id=g1_id, created_at=now, updated_at=now
    )
    stock2 = Stock(
        id=uuid4(), ticker="000660", group_id=g2_id, created_at=now, updated_at=now
    )
    stock3 = Stock(
        id=uuid4(), ticker="035420", group_id=g3_id, created_at=now, updated_at=now
    )

    # Holdings with different values:
    # SmallGroup: 100,000
    # BigGroup: 500,000
    # MediumGroup: 300,000
    holdings = [
        (
            group1,
            StockHoldingWithPrice(
                stock=stock1,
                quantity=Decimal("1"),
                price=Decimal("100000"),
                currency="KRW",
                name="Small Stock",
                value_krw=Decimal("100000"),
            ),
        ),
        (
            group2,
            StockHoldingWithPrice(
                stock=stock2,
                quantity=Decimal("5"),
                price=Decimal("100000"),
                currency="KRW",
                name="Big Stock",
                value_krw=Decimal("500000"),
            ),
        ),
        (
            group3,
            StockHoldingWithPrice(
                stock=stock3,
                quantity=Decimal("3"),
                price=Decimal("100000"),
                currency="KRW",
                name="Medium Stock",
                value_krw=Decimal("300000"),
            ),
        ),
    ]

    summary = PortfolioSummary(holdings=holdings, total_value=Decimal("900000"))
    console = Console(record=True, width=140)

    render_dashboard(console, summary)

    output = console.export_text()

    # Find the positions of group names in the output
    # Group Summary should show: BigGroup (500k) > MediumGroup (300k) > SmallGroup (100k)
    big_pos = output.find("BigGroup")
    medium_pos = output.find("MediumGroup")
    small_pos = output.find("SmallGroup")

    # All groups should appear
    assert big_pos != -1, "BigGroup not found in output"
    assert medium_pos != -1, "MediumGroup not found in output"
    assert small_pos != -1, "SmallGroup not found in output"

    # Groups should appear in descending value order in Group Summary section
    # First find where Group Summary section starts
    summary_start = output.find("Group Summary")
    assert summary_start != -1, "Group Summary section not found"

    # Get positions after Group Summary section
    output_after_summary = output[summary_start:]
    big_pos_summary = output_after_summary.find("BigGroup")
    medium_pos_summary = output_after_summary.find("MediumGroup")
    small_pos_summary = output_after_summary.find("SmallGroup")

    assert big_pos_summary < medium_pos_summary < small_pos_summary, (
        f"Groups not sorted by value descending in Group Summary. "
        f"BigGroup at {big_pos_summary}, MediumGroup at {medium_pos_summary}, "
        f"SmallGroup at {small_pos_summary}"
    )
