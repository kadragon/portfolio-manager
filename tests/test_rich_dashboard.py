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
