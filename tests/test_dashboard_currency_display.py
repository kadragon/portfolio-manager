"""Tests for dashboard currency display."""

from decimal import Decimal
from uuid import uuid4

from rich.console import Console

from portfolio_manager.cli.rich_app import render_dashboard
from portfolio_manager.models import Group, Stock
from portfolio_manager.services.portfolio_service import (
    PortfolioSummary,
    StockHoldingWithPrice,
)


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
            ),
        ),
    ]
    summary = PortfolioSummary(holdings=holdings, total_value=Decimal("700000"))

    render_dashboard(console, summary)

    output = console.export_text()
    assert "₩70,000" in output
    assert "₩700,000" in output


def test_dashboard_displays_usd_for_overseas_stocks():
    """해외 주식은 $ 기호로 표시한다."""
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
            ),
        ),
    ]
    summary = PortfolioSummary(holdings=holdings, total_value=Decimal("750.0"))

    render_dashboard(console, summary)

    output = console.export_text()
    assert "$150" in output
    assert "$750" in output


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
            ),
        ),
        (
            group2,
            StockHoldingWithPrice(
                stock=stock2,
                quantity=Decimal("5"),
                price=Decimal("150.0"),
                currency="USD",
            ),
        ),
    ]
    summary = PortfolioSummary(holdings=holdings, total_value=Decimal("700750.0"))

    render_dashboard(console, summary)

    output = console.export_text()
    assert "₩70,000" in output
    assert "$150" in output
