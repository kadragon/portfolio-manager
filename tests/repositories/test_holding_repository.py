"""Tests for holding repository."""

from decimal import Decimal
from uuid import uuid4
from unittest.mock import Mock

from portfolio_manager.repositories.holding_repository import HoldingRepository


def test_holding_repository_creates_holding_with_decimal_quantity():
    """Should create a holding with decimal quantity."""
    holding_id = uuid4()
    account_id = uuid4()
    stock_id = uuid4()
    response = Mock()
    response.data = [
        {
            "id": str(holding_id),
            "account_id": str(account_id),
            "stock_id": str(stock_id),
            "quantity": "10.75",
            "created_at": "2026-01-03T00:00:00",
            "updated_at": "2026-01-03T00:00:00",
        }
    ]

    client = Mock()
    client.table.return_value.insert.return_value.execute.return_value = response

    repository = HoldingRepository(client)
    holding = repository.create(
        account_id=account_id, stock_id=stock_id, quantity=Decimal("10.75")
    )

    client.table.assert_called_once_with("holdings")
    client.table.return_value.insert.assert_called_once_with(
        {
            "account_id": str(account_id),
            "stock_id": str(stock_id),
            "quantity": "10.75",
        }
    )
    assert holding.quantity == Decimal("10.75")


def test_holding_repository_deletes_by_id():
    """Should delete holding by id."""
    holding_id = uuid4()
    client = Mock()
    client.table.return_value.delete.return_value.eq.return_value.execute.return_value = Mock()

    repository = HoldingRepository(client)
    repository.delete(holding_id)

    client.table.assert_called_once_with("holdings")
    client.table.return_value.delete.assert_called_once()
    client.table.return_value.delete.return_value.eq.assert_called_once_with(
        "id", str(holding_id)
    )


def test_aggregate_holdings_by_stock():
    """모든 계좌에서 동일 주식의 보유 수량을 합산한다."""
    # Given: 서버 사이드 집계 결과가 반환됨
    stock_id = uuid4()

    response = Mock()
    response.data = [
        {
            "stock_id": str(stock_id),
            "quantity": "15",
        },
    ]

    client = Mock()
    client.rpc.return_value.execute.return_value = response
    client.table.side_effect = AssertionError("table select should not be used")

    repository = HoldingRepository(client)

    # When: 주식별로 합산된 보유 수량을 조회
    aggregated = repository.get_aggregated_holdings_by_stock()

    # Then: 해당 주식의 수량이 15로 합산됨
    client.rpc.assert_called_once_with("aggregate_holdings_by_stock")
    assert stock_id in aggregated
    assert aggregated[stock_id] == Decimal("15")


def test_holding_repository_reads_decimal_quantity():
    """Should parse decimal quantities when listing holdings."""
    account_id = uuid4()
    response = Mock()
    response.data = [
        {
            "id": str(uuid4()),
            "account_id": str(account_id),
            "stock_id": str(uuid4()),
            "quantity": "3.1415",
            "created_at": "2026-01-03T00:00:00",
            "updated_at": "2026-01-03T00:00:00",
        }
    ]
    client = Mock()
    client.table.return_value.select.return_value.eq.return_value.execute.return_value = response

    repository = HoldingRepository(client)
    holdings = repository.list_by_account(account_id)

    assert holdings[0].quantity == Decimal("3.1415")
