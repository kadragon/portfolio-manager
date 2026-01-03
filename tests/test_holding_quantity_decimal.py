"""Tests for decimal quantities in holdings."""

from decimal import Decimal
from uuid import uuid4
from unittest.mock import Mock

from portfolio_manager.repositories.holding_repository import HoldingRepository


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
