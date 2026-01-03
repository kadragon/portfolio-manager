"""Tests for account repository."""

from decimal import Decimal
from uuid import uuid4
from unittest.mock import Mock

from portfolio_manager.repositories.account_repository import AccountRepository


def test_account_repository_creates_account_with_cash_balance():
    """Should create an account with cash balance."""
    account_id = uuid4()
    response = Mock()
    response.data = [
        {
            "id": str(account_id),
            "name": "Main Account",
            "cash_balance": "100000.50",
            "created_at": "2026-01-03T00:00:00",
            "updated_at": "2026-01-03T00:00:00",
        }
    ]

    client = Mock()
    client.table.return_value.insert.return_value.execute.return_value = response

    repository = AccountRepository(client)
    account = repository.create(name="Main Account", cash_balance=Decimal("100000.50"))

    client.table.assert_called_once_with("accounts")
    client.table.return_value.insert.assert_called_once_with(
        {"name": "Main Account", "cash_balance": "100000.50"}
    )
    assert account.name == "Main Account"
    assert account.cash_balance == Decimal("100000.50")
