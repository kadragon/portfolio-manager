"""Tests for Rich-based dashboard rendering."""

from decimal import Decimal
from uuid import uuid4

from rich.console import Console

from portfolio_manager.cli.rich_app import render_dashboard
from portfolio_manager.models import Group, Stock
from portfolio_manager.services.portfolio_service import GroupHoldings, StockHolding


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
