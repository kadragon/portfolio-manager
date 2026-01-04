"""Tests for DepositRepository."""

from datetime import date
from unittest.mock import MagicMock


def test_get_first_deposit_date_returns_earliest():
    """Should return the earliest deposit date."""
    from portfolio_manager.repositories.deposit_repository import DepositRepository

    mock_client = MagicMock()
    mock_client.table.return_value.select.return_value.order.return_value.limit.return_value.execute.return_value.data = [
        {
            "id": "123",
            "amount": "1000000",
            "deposit_date": "2024-01-15",
            "note": None,
            "created_at": "2024-01-15T00:00:00",
            "updated_at": "2024-01-15T00:00:00",
        }
    ]

    repo = DepositRepository(mock_client)
    result = repo.get_first_deposit_date()

    assert result == date(2024, 1, 15)
    mock_client.table.assert_called_with("deposits")


def test_get_first_deposit_date_returns_none_when_empty():
    """Should return None when no deposits exist."""
    from portfolio_manager.repositories.deposit_repository import DepositRepository

    mock_client = MagicMock()
    mock_client.table.return_value.select.return_value.order.return_value.limit.return_value.execute.return_value.data = []

    repo = DepositRepository(mock_client)
    result = repo.get_first_deposit_date()

    assert result is None
