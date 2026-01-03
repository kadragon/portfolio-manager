"""Tests for holding repository delete."""

from uuid import uuid4
from unittest.mock import Mock

from portfolio_manager.repositories.holding_repository import HoldingRepository


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
