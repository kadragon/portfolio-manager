"""Test holding aggregation across accounts."""

from decimal import Decimal
from unittest.mock import Mock
from uuid import uuid4

from portfolio_manager.repositories.holding_repository import HoldingRepository


def test_aggregate_holdings_by_stock():
    """모든 계좌에서 동일 주식의 보유 수량을 합산한다."""
    # Given: 두 계좌에서 같은 주식을 보유
    account1_id = uuid4()
    account2_id = uuid4()
    stock_id = uuid4()

    response = Mock()
    response.data = [
        {
            "id": str(uuid4()),
            "account_id": str(account1_id),
            "stock_id": str(stock_id),
            "quantity": "10",
            "created_at": "2026-01-03T00:00:00",
            "updated_at": "2026-01-03T00:00:00",
        },
        {
            "id": str(uuid4()),
            "account_id": str(account2_id),
            "stock_id": str(stock_id),
            "quantity": "5",
            "created_at": "2026-01-03T00:00:00",
            "updated_at": "2026-01-03T00:00:00",
        },
    ]

    client = Mock()
    client.table.return_value.select.return_value.execute.return_value = response

    repository = HoldingRepository(client)

    # When: 주식별로 합산된 보유 수량을 조회
    aggregated = repository.get_aggregated_holdings_by_stock()

    # Then: 해당 주식의 수량이 15로 합산됨
    assert stock_id in aggregated
    assert aggregated[stock_id] == Decimal("15")
