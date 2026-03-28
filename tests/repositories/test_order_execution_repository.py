"""Tests for order execution repository."""

from portfolio_manager.models.order_execution import OrderExecutionRecord
from portfolio_manager.repositories.order_execution_repository import (
    OrderExecutionRepository,
)


def test_order_execution_repository_creates_and_deserializes():
    repo = OrderExecutionRepository()
    record = repo.create(
        ticker="005930",
        side="buy",
        quantity=10,
        currency="KRW",
        exchange=None,
        status="success",
        message="주문 전송 완료",
        raw_response={"rt_cd": "0", "msg1": "정상처리"},
    )

    assert isinstance(record, OrderExecutionRecord)
    assert record.ticker == "005930"
    assert record.side == "buy"
    assert record.quantity == 10
    assert record.currency == "KRW"
    assert record.exchange is None
    assert record.status == "success"
    assert record.message == "주문 전송 완료"
    assert record.raw_response == {"rt_cd": "0", "msg1": "정상처리"}


def test_order_execution_repository_lists_recent_in_descending_order():
    repo = OrderExecutionRepository()
    repo.create(
        ticker="005930",
        side="buy",
        quantity=10,
        currency="KRW",
        status="success",
        message="first",
    )
    repo.create(
        ticker="AAPL",
        side="sell",
        quantity=5,
        currency="USD",
        exchange="NASD",
        status="failed",
        message="second",
        raw_response={"rt_cd": "1"},
    )

    records = repo.list_recent(limit=10)

    assert len(records) == 2
    assert all(isinstance(r, OrderExecutionRecord) for r in records)
    # Most recent first
    assert records[0].ticker == "AAPL"
    assert records[0].exchange == "NASD"
    assert records[0].raw_response == {"rt_cd": "1"}
    assert records[1].ticker == "005930"
    assert records[1].raw_response is None
