"""Tests for compute_group_summary."""

from datetime import datetime
from decimal import Decimal
from uuid import uuid4

from portfolio_manager.models import Group, Stock
from portfolio_manager.services.group_summary import compute_group_summary
from portfolio_manager.services.portfolio_service import (
    PortfolioSummary,
    StockHoldingWithPrice,
)


def _group(name: str, target_pct: float = 0.0) -> Group:
    return Group(
        id=uuid4(),
        name=name,
        target_percentage=target_pct,
        created_at=datetime(2024, 1, 1),
        updated_at=datetime(2024, 1, 1),
    )


def _holding(stock: Stock, value_krw: Decimal) -> StockHoldingWithPrice:
    return StockHoldingWithPrice(
        stock=stock,
        quantity=Decimal("1"),
        price=value_krw,
        currency="KRW",
        name=stock.ticker,
        value_krw=value_krw,
    )


def _summary(
    holdings: list[tuple[Group, StockHoldingWithPrice]],
    total_stock_value: Decimal,
) -> PortfolioSummary:
    return PortfolioSummary(
        holdings=holdings,
        total_value=total_stock_value,
        total_stock_value=total_stock_value,
    )


def test_rows_sorted_by_total_descending():
    g1 = _group("A", target_pct=50.0)
    g2 = _group("B", target_pct=50.0)
    s1 = Stock(
        id=uuid4(), ticker="T1", group_id=g1.id, created_at=None, updated_at=None
    )  # type: ignore[arg-type]
    s2 = Stock(
        id=uuid4(), ticker="T2", group_id=g2.id, created_at=None, updated_at=None
    )  # type: ignore[arg-type]

    summary = _summary(
        holdings=[
            (g1, _holding(s1, Decimal("300000"))),
            (g2, _holding(s2, Decimal("700000"))),
        ],
        total_stock_value=Decimal("1000000"),
    )

    rows = compute_group_summary(summary)

    assert rows[0].group.name == "B"
    assert rows[1].group.name == "A"


def test_actual_pct_uses_total_stock_value_as_denominator():
    g = _group("A", target_pct=0.0)
    s = Stock(id=uuid4(), ticker="T", group_id=g.id, created_at=None, updated_at=None)  # type: ignore[arg-type]

    summary = _summary(
        holdings=[(g, _holding(s, Decimal("400000")))],
        total_stock_value=Decimal("1000000"),
    )

    rows = compute_group_summary(summary)

    assert rows[0].actual_pct == Decimal("40")


def test_diff_pct_and_diff_val():
    g = _group("A", target_pct=50.0)
    s = Stock(id=uuid4(), ticker="T", group_id=g.id, created_at=None, updated_at=None)  # type: ignore[arg-type]

    summary = _summary(
        holdings=[(g, _holding(s, Decimal("600000")))],
        total_stock_value=Decimal("1000000"),
    )

    rows = compute_group_summary(summary)

    assert rows[0].diff_pct == Decimal("10")  # 60% - 50%
    assert rows[0].diff_val == Decimal("100000")  # 600k - 1000k*50%/100


def test_zero_total_stock_value_returns_zero_pct():
    g = _group("A", target_pct=30.0)
    s = Stock(id=uuid4(), ticker="T", group_id=g.id, created_at=None, updated_at=None)  # type: ignore[arg-type]

    summary = _summary(
        holdings=[(g, _holding(s, Decimal("0")))],
        total_stock_value=Decimal("0"),
    )

    rows = compute_group_summary(summary)

    assert rows[0].actual_pct == Decimal("0")


def test_all_groups_includes_zero_holding_groups():
    g1 = _group("Held", target_pct=60.0)
    g2 = _group("Empty", target_pct=40.0)
    s = Stock(id=uuid4(), ticker="T", group_id=g1.id, created_at=None, updated_at=None)  # type: ignore[arg-type]

    summary = _summary(
        holdings=[(g1, _holding(s, Decimal("1000000")))],
        total_stock_value=Decimal("1000000"),
    )

    rows = compute_group_summary(summary, all_groups=[g1, g2])

    assert len(rows) == 2
    empty_row = next(r for r in rows if r.group.name == "Empty")
    assert empty_row.total == Decimal("0")
    assert empty_row.actual_pct == Decimal("0")
    assert empty_row.diff_pct == Decimal("-40")


def test_value_krw_fallback_to_value():
    """Holdings with value_krw=None use value (quantity * price) instead."""
    g = _group("A")
    s = Stock(id=uuid4(), ticker="T", group_id=g.id, created_at=None, updated_at=None)  # type: ignore[arg-type]
    h = StockHoldingWithPrice(
        stock=s,
        quantity=Decimal("10"),
        price=Decimal("50000"),
        currency="USD",
        name="T",
        value_krw=None,  # USD holding — no KRW conversion
    )

    summary = _summary(
        holdings=[(g, h)],
        total_stock_value=Decimal("500000"),
    )

    rows = compute_group_summary(summary)

    assert rows[0].total == Decimal("500000")  # 10 * 50000
