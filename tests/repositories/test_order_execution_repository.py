"""Tests for order execution repository."""

import json
from uuid import uuid4
from unittest.mock import Mock

from portfolio_manager.models.order_execution import OrderExecutionRecord
from portfolio_manager.repositories.order_execution_repository import (
    OrderExecutionRepository,
)


def test_order_execution_repository_creates_and_deserializes():
    """Should create an order execution record and deserialize it."""

    execution_id = uuid4()

    response = Mock()

    response.data = [
        {
            "id": str(execution_id),
            "ticker": "005930",
            "side": "buy",
            "quantity": 10,
            "currency": "KRW",
            "exchange": None,
            "status": "success",
            "message": "주문 전송 완료",
            "raw_response": {"rt_cd": "0", "msg1": "정상처리"},
            "created_at": "2026-02-09T00:00:00",
        }
    ]

    client = Mock()

    client.table.return_value.insert.return_value.execute.return_value = response

    repository = OrderExecutionRepository(client)

    record = repository.create(
        ticker="005930",
        side="buy",
        quantity=10,
        currency="KRW",
        exchange=None,
        status="success",
        message="주문 전송 완료",
        raw_response={"rt_cd": "0", "msg1": "정상처리"},
    )

    client.table.assert_called_once_with("order_executions")

    client.table.return_value.insert.assert_called_once_with(
        {
            "ticker": "005930",
            "side": "buy",
            "quantity": 10,
            "currency": "KRW",
            "exchange": None,
            "status": "success",
            "message": "주문 전송 완료",
            "raw_response": json.dumps({"rt_cd": "0", "msg1": "정상처리"}),
        }
    )

    assert isinstance(record, OrderExecutionRecord)

    assert record.id == execution_id

    assert record.ticker == "005930"

    assert record.side == "buy"

    assert record.quantity == 10

    assert record.currency == "KRW"

    assert record.exchange is None

    assert record.status == "success"

    assert record.message == "주문 전송 완료"

    assert record.raw_response == {"rt_cd": "0", "msg1": "정상처리"}


def test_order_execution_repository_lists_recent_in_descending_order():
    """Should list recent executions ordered by created_at descending."""

    id_1 = uuid4()
    id_2 = uuid4()

    response = Mock()

    response.data = [
        {
            "id": str(id_2),
            "ticker": "AAPL",
            "side": "sell",
            "quantity": 5,
            "currency": "USD",
            "exchange": "NASD",
            "status": "failed",
            "message": "잔고부족",
            "raw_response": {"rt_cd": "1", "msg_cd": "APBK1234"},
            "created_at": "2026-02-09T10:00:00",
        },
        {
            "id": str(id_1),
            "ticker": "005930",
            "side": "buy",
            "quantity": 10,
            "currency": "KRW",
            "exchange": None,
            "status": "success",
            "message": "주문 전송 완료",
            "raw_response": None,
            "created_at": "2026-02-09T09:00:00",
        },
    ]

    client = Mock()

    client.table.return_value.select.return_value.order.return_value.limit.return_value.execute.return_value = response

    repository = OrderExecutionRepository(client)

    records = repository.list_recent(limit=10)

    client.table.assert_called_once_with("order_executions")

    client.table.return_value.select.assert_called_once_with("*")

    client.table.return_value.select.return_value.order.assert_called_once_with(
        "created_at", desc=True
    )

    client.table.return_value.select.return_value.order.return_value.limit.assert_called_once_with(
        10
    )

    assert len(records) == 2

    assert all(isinstance(r, OrderExecutionRecord) for r in records)

    assert records[0].id == id_2
    assert records[0].ticker == "AAPL"
    assert records[0].exchange == "NASD"
    assert records[0].status == "failed"
    assert records[0].raw_response == {"rt_cd": "1", "msg_cd": "APBK1234"}

    assert records[1].id == id_1
    assert records[1].ticker == "005930"
    assert records[1].raw_response is None
