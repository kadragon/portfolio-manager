"""Tests for Rich-based dashboard rendering."""

from datetime import date
from decimal import Decimal
from uuid import uuid4

from rich.console import Console

from portfolio_manager.cli.app import render_dashboard
from portfolio_manager.models import Group, Stock
from portfolio_manager.services.portfolio_service import (
    GroupHoldings,
    PortfolioSummary,
    StockHolding,
    StockHoldingWithPrice,
)


def test_render_dashboard_shows_groups_and_stocks():
    """메인 메뉴에서 그룹별 주식 대시보드를 표시한다."""
    console = Console(record=True, width=120)

    # Given: 그룹과 주식 데이터
    group1 = Group(
        id=uuid4(),
        name="Tech",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )
    stock1 = Stock(
        id=uuid4(),
        ticker="AAPL",
        group_id=group1.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )
    stock2 = Stock(
        id=uuid4(),
        ticker="GOOGL",
        group_id=group1.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )

    group_holdings = [
        GroupHoldings(
            group=group1,
            stock_holdings=[
                StockHolding(stock=stock1, quantity=Decimal("10")),
                StockHolding(stock=stock2, quantity=Decimal("5")),
            ],
        )
    ]

    # When: 대시보드를 렌더링
    render_dashboard(console, group_holdings)

    # Then: 그룹명과 주식 정보가 표시됨
    output = console.export_text()
    assert "Tech" in output
    assert "AAPL" in output
    assert "GOOGL" in output
    assert "10" in output
    assert "5" in output


def test_render_dashboard_shows_message_when_no_groups():
    """그룹이 없을 때 안내 메시지를 표시한다."""
    console = Console(record=True, width=120)

    # Given: 빈 그룹 목록
    group_holdings = []

    # When: 대시보드를 렌더링
    render_dashboard(console, group_holdings)

    # Then: 안내 메시지가 표시됨
    output = console.export_text()
    assert "No groups" in output or "no groups" in output


def test_render_dashboard_shows_groups_without_stocks():
    """주식이 없는 그룹도 표시한다."""
    console = Console(record=True, width=120)

    # Given: 주식이 없는 그룹
    group1 = Group(
        id=uuid4(),
        name="Empty Group",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )

    group_holdings = [
        GroupHoldings(
            group=group1,
            stock_holdings=[],
        )
    ]

    # When: 대시보드를 렌더링
    render_dashboard(console, group_holdings)

    # Then: 그룹명이 표시됨
    output = console.export_text()
    assert "Empty Group" in output


def test_render_dashboard_shows_all_stocks_in_single_table():
    """단일 테이블로 모든 주식과 그룹명을 표시한다."""
    console = Console(record=True, width=120)

    # Given: 여러 그룹과 주식
    group1 = Group(
        id=uuid4(),
        name="Tech",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )
    group2 = Group(
        id=uuid4(),
        name="Finance",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )
    stock1 = Stock(
        id=uuid4(),
        ticker="AAPL",
        group_id=group1.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )
    stock2 = Stock(
        id=uuid4(),
        ticker="JPM",
        group_id=group2.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )

    group_holdings = [
        GroupHoldings(
            group=group1,
            stock_holdings=[StockHolding(stock=stock1, quantity=Decimal("10"))],
        ),
        GroupHoldings(
            group=group2,
            stock_holdings=[StockHolding(stock=stock2, quantity=Decimal("5"))],
        ),
    ]

    # When: 대시보드를 렌더링
    render_dashboard(console, group_holdings)

    # Then: 단일 테이블에 모든 주식과 그룹이 표시됨
    output = console.export_text()
    assert "Tech" in output
    assert "Finance" in output
    assert "AAPL" in output
    assert "JPM" in output
    # 테이블 제목이 하나만 있어야 함 (여러 테이블이 아님)
    assert output.count("📊") == 1 or "Portfolio" in output


def test_render_dashboard_shows_prices_and_values():
    """가격과 평가금액을 표시한다."""
    console = Console(record=True, width=120)

    # Given: Portfolio summary with prices
    group = Group(
        id=uuid4(),
        name="Tech",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )
    stock1 = Stock(
        id=uuid4(),
        ticker="AAPL",
        group_id=group.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )
    stock2 = Stock(
        id=uuid4(),
        ticker="GOOGL",
        group_id=group.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )

    holdings = [
        (
            group,
            StockHoldingWithPrice(
                stock=stock1,
                quantity=Decimal("10"),
                price=Decimal("150.0"),
                currency="USD",
                name="Apple Inc.",
            ),
        ),
        (
            group,
            StockHoldingWithPrice(
                stock=stock2,
                quantity=Decimal("5"),
                price=Decimal("100.0"),
                currency="USD",
                name="Google",
            ),
        ),
    ]
    summary = PortfolioSummary(
        holdings=holdings,
        total_value=Decimal("2600000.0"),
        total_stock_value=Decimal("2600000.0"),
        total_assets=Decimal("2600000.0"),
    )

    # When: 대시보드를 렌더링
    render_dashboard(console, summary)

    # Then: 가격, 평가금액, 총계가 표시됨
    output = console.export_text()
    assert "AAPL" in output
    assert "GOOGL" in output
    assert "150" in output  # price
    assert "1,500" in output  # value (10 × 150)
    assert "100" in output  # price
    assert "500" in output  # value (5 × 100)
    assert "2,600,000" in output  # total value


def test_dashboard_displays_group_totals_and_rebalance_info():
    """그룹별 합계, 비중, 목표 대비 매수/매도 정보를 표시한다."""
    console = Console(record=True, width=140)

    group1 = Group(
        id=uuid4(),
        name="국내성장",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
        target_percentage=50.0,
    )
    group2 = Group(
        id=uuid4(),
        name="해외성장",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
        target_percentage=50.0,
    )
    stock1 = Stock(
        id=uuid4(),
        ticker="005930",
        group_id=group1.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )
    stock2 = Stock(
        id=uuid4(),
        ticker="AAPL",
        group_id=group2.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )

    holdings = [
        (
            group1,
            StockHoldingWithPrice(
                stock=stock1,
                quantity=Decimal("10"),
                price=Decimal("60000"),
                currency="KRW",
                name="삼성전자",
            ),
        ),
        (
            group2,
            StockHoldingWithPrice(
                stock=stock2,
                quantity=Decimal("10"),
                price=Decimal("40000"),
                currency="KRW",
                name="Apple Inc.",
            ),
        ),
    ]
    summary = PortfolioSummary(holdings=holdings, total_value=Decimal("1000000"))

    render_dashboard(console, summary)

    output = console.export_text()
    assert "Group Summary" in output
    assert "₩600,000" in output
    assert "60.0%" in output
    assert "50.0%" in output
    assert "+10.0%" in output
    assert "Amount" in output
    assert "🔴 Sell" in output
    assert "🟢 Buy" in output
    assert "Sell ₩100,000" not in output
    assert "Buy ₩100,000" not in output
    assert "₩100,000" in output


def test_dashboard_displays_hold_as_dash():
    """목표 비중과 동일하면 Hold 대신 대시를 표시한다."""
    console = Console(record=True, width=140)

    group = Group(
        id=uuid4(),
        name="균형",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
        target_percentage=100.0,
    )
    stock = Stock(
        id=uuid4(),
        ticker="005930",
        group_id=group.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )

    holdings = [
        (
            group,
            StockHoldingWithPrice(
                stock=stock,
                quantity=Decimal("10"),
                price=Decimal("10000"),
                currency="KRW",
                name="삼성전자",
            ),
        ),
    ]
    summary = PortfolioSummary(holdings=holdings, total_value=Decimal("100000"))

    render_dashboard(console, summary)

    output = console.export_text()
    assert "Hold" not in output
    assert "Amount" in output
    assert " - " in output


def test_dashboard_colors_diff_percent_to_match_action():
    """Diff %를 매수/매도와 동일한 색상으로 표시한다."""
    console = Console(
        record=True, width=140, force_terminal=True, color_system="standard"
    )

    group1 = Group(
        id=uuid4(),
        name="국내성장",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
        target_percentage=50.0,
    )
    group2 = Group(
        id=uuid4(),
        name="해외성장",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
        target_percentage=50.0,
    )
    stock1 = Stock(
        id=uuid4(),
        ticker="005930",
        group_id=group1.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )
    stock2 = Stock(
        id=uuid4(),
        ticker="AAPL",
        group_id=group2.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )

    holdings = [
        (
            group1,
            StockHoldingWithPrice(
                stock=stock1,
                quantity=Decimal("10"),
                price=Decimal("60000"),
                currency="KRW",
                name="삼성전자",
            ),
        ),
        (
            group2,
            StockHoldingWithPrice(
                stock=stock2,
                quantity=Decimal("10"),
                price=Decimal("40000"),
                currency="KRW",
                name="Apple Inc.",
            ),
        ),
    ]
    summary = PortfolioSummary(holdings=holdings, total_value=Decimal("1000000"))

    render_dashboard(console, summary)

    output = console.export_text(styles=True)
    assert "\x1b[31m+10.0%" in output
    assert "\x1b[32m-10.0%" in output


def test_dashboard_displays_krw_for_domestic_stocks():
    """국내 주식은 ₩ 기호로 표시한다."""
    console = Console(record=True, width=120)

    group = Group(
        id=uuid4(),
        name="국내성장",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )
    stock = Stock(
        id=uuid4(),
        ticker="005930",
        group_id=group.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )

    holdings = [
        (
            group,
            StockHoldingWithPrice(
                stock=stock,
                quantity=Decimal("10"),
                price=Decimal("70000"),
                currency="KRW",
                name="삼성전자",
            ),
        ),
    ]
    summary = PortfolioSummary(holdings=holdings, total_value=Decimal("700000"))

    render_dashboard(console, summary)

    output = console.export_text()
    assert "₩70,000" in output
    assert "₩700,000" in output


def test_dashboard_displays_usd_for_overseas_stocks():
    """해외 주식은 가격은 $, 평가는 ₩로 표시한다."""
    console = Console(record=True, width=120)

    group = Group(
        id=uuid4(),
        name="해외성장",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )
    stock = Stock(
        id=uuid4(),
        ticker="AAPL",
        group_id=group.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )

    holdings = [
        (
            group,
            StockHoldingWithPrice(
                stock=stock,
                quantity=Decimal("5"),
                price=Decimal("150.0"),
                currency="USD",
                name="Apple Inc.",
                value_krw=Decimal("975000.0"),
            ),
        ),
    ]
    summary = PortfolioSummary(holdings=holdings, total_value=Decimal("975000.0"))

    render_dashboard(console, summary)

    output = console.export_text()
    assert "$150" in output
    assert "₩975,000" in output


def test_dashboard_rounds_overseas_quantity_to_integer():
    """해외 주식 수량은 소수점 첫째 자리에서 반올림한다."""
    console = Console(record=True, width=120)

    group = Group(
        id=uuid4(),
        name="해외배당",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )
    stock = Stock(
        id=uuid4(),
        ticker="VYM",
        group_id=group.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )

    holdings = [
        (
            group,
            StockHoldingWithPrice(
                stock=stock,
                quantity=Decimal("33.5"),
                price=Decimal("0"),
                currency="USD",
                name="Vanguard High Dividend Yield ETF",
                value_krw=Decimal("0"),
            ),
        ),
    ]
    summary = PortfolioSummary(holdings=holdings, total_value=Decimal("0"))

    render_dashboard(console, summary)

    output = console.export_text()
    assert "33.5" not in output
    assert "34" in output


def test_dashboard_displays_mixed_currencies():
    """국내/해외 주식이 섞여있으면 각각 다른 기호로 표시한다."""
    console = Console(record=True, width=120)

    group1 = Group(
        id=uuid4(),
        name="국내성장",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )
    stock1 = Stock(
        id=uuid4(),
        ticker="005930",
        group_id=group1.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )

    group2 = Group(
        id=uuid4(),
        name="해외성장",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )
    stock2 = Stock(
        id=uuid4(),
        ticker="AAPL",
        group_id=group2.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )

    holdings = [
        (
            group1,
            StockHoldingWithPrice(
                stock=stock1,
                quantity=Decimal("10"),
                price=Decimal("70000"),
                currency="KRW",
                name="삼성전자",
            ),
        ),
        (
            group2,
            StockHoldingWithPrice(
                stock=stock2,
                quantity=Decimal("5"),
                price=Decimal("150.0"),
                currency="USD",
                name="Apple Inc.",
                value_krw=Decimal("975000.0"),
            ),
        ),
    ]
    summary = PortfolioSummary(holdings=holdings, total_value=Decimal("1675000.0"))

    render_dashboard(console, summary)

    output = console.export_text()
    assert "₩70,000" in output
    assert "$150" in output
    assert "₩975,000" in output


def test_dashboard_displays_stock_name():
    """대시보드에 주식명을 표시한다."""
    console = Console(record=True, width=120)

    group = Group(
        id=uuid4(),
        name="국내성장",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )
    stock = Stock(
        id=uuid4(),
        ticker="005930",
        group_id=group.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )

    holdings = [
        (
            group,
            StockHoldingWithPrice(
                stock=stock,
                quantity=Decimal("10"),
                price=Decimal("70000"),
                currency="KRW",
                name="삼성전자",
            ),
        ),
    ]
    summary = PortfolioSummary(holdings=holdings, total_value=Decimal("700000"))

    render_dashboard(console, summary)

    output = console.export_text()
    assert "삼성전자" in output


def test_dashboard_uses_ticker_when_name_missing():
    """주식명이 없으면 티커를 대신 표시한다."""
    console = Console(record=True, width=120)

    group = Group(
        id=uuid4(),
        name="해외배당",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )
    stock = Stock(
        id=uuid4(),
        ticker="VYM",
        group_id=group.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )

    holdings = [
        (
            group,
            StockHoldingWithPrice(
                stock=stock,
                quantity=Decimal("10"),
                price=Decimal("100.0"),
                currency="USD",
                name="",
                value_krw=Decimal("1300000.0"),
            ),
        ),
    ]
    summary = PortfolioSummary(holdings=holdings, total_value=Decimal("1300000.0"))

    render_dashboard(console, summary)

    output = console.export_text()
    assert output.count("VYM") >= 2


def test_dashboard_truncates_long_stock_names():
    """긴 주식명은 잘라서 표시한다."""
    console = Console(record=True, width=120)

    group = Group(
        id=uuid4(),
        name="해외성장",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )
    stock = Stock(
        id=uuid4(),
        ticker="AAPL",
        group_id=group.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )

    holdings = [
        (
            group,
            StockHoldingWithPrice(
                stock=stock,
                quantity=Decimal("5"),
                price=Decimal("150.0"),
                currency="USD",
                name="Apple Inc. Corporation Limited",
                value_krw=Decimal("975000.0"),
            ),
        ),
    ]
    summary = PortfolioSummary(holdings=holdings, total_value=Decimal("975000.0"))

    render_dashboard(console, summary)

    output = console.export_text()
    # 25자로 잘린 이름 확인 (Apple Inc.는 포함됨)
    assert "Apple Inc." in output
    # 전체 이름은 표시되지 않음
    assert "Apple Inc. Corporation Limited" not in output


def test_dashboard_displays_annualized_return_rate():
    """연환산 수익률을 표시한다."""
    console = Console(record=True, width=120)

    group = Group(
        id=uuid4(),
        name="Tech",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )
    stock = Stock(
        id=uuid4(),
        ticker="AAPL",
        group_id=group.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )

    holdings = [
        (
            group,
            StockHoldingWithPrice(
                stock=stock,
                quantity=Decimal("10"),
                price=Decimal("150.0"),
                currency="USD",
                name="Apple Inc.",
                value_krw=Decimal("1950000"),
            ),
        ),
    ]
    summary = PortfolioSummary(
        holdings=holdings,
        total_value=Decimal("1950000"),
        total_stock_value=Decimal("1950000"),
        total_cash_balance=Decimal("0"),
        total_assets=Decimal("1950000"),
        total_invested=Decimal("1500000"),
        return_rate=Decimal("30.0"),
        first_deposit_date=date(2024, 1, 15),
        annualized_return_rate=Decimal("25.5"),
    )

    render_dashboard(console, summary)

    output = console.export_text()
    assert "Annualized" in output
    assert "25.5" in output


def test_dashboard_displays_investment_summary_panel():
    """투자 요약을 Panel로 표시한다."""
    console = Console(record=True, width=120)

    group = Group(
        id=uuid4(),
        name="Tech",
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )
    stock = Stock(
        id=uuid4(),
        ticker="AAPL",
        group_id=group.id,
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )

    holdings = [
        (
            group,
            StockHoldingWithPrice(
                stock=stock,
                quantity=Decimal("10"),
                price=Decimal("150.0"),
                currency="USD",
                name="Apple Inc.",
                value_krw=Decimal("1950000"),
            ),
        ),
    ]
    summary = PortfolioSummary(
        holdings=holdings,
        total_value=Decimal("1950000"),
        total_stock_value=Decimal("1950000"),
        total_cash_balance=Decimal("50000"),
        total_assets=Decimal("2000000"),
        total_invested=Decimal("1500000"),
        return_rate=Decimal("33.33"),
    )

    render_dashboard(console, summary)

    output = console.export_text()
    # Panel 형태인지 확인 (테두리 존재)
    assert "━" in output or "─" in output or "═" in output
    # 필수 정보가 표시되는지 확인
    assert "1,950,000" in output  # Stock Value
    assert "50,000" in output  # Cash Balance
    assert "2,000,000" in output  # Total Assets
    assert "1,500,000" in output  # Total Invested
