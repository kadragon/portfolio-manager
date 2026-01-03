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
