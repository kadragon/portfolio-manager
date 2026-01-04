"""Tests for account repository."""

from decimal import Decimal
from uuid import uuid4
from unittest.mock import Mock, MagicMock

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


def test_account_repository_deletes_holdings_before_account():
    """Should delete holdings before deleting the account."""
    account_id = uuid4()
    client = Mock()
    client.table.return_value.delete.return_value.eq.return_value.execute.return_value = Mock()
    repository = AccountRepository(client)
    holding_repo = MagicMock()

    repository.delete_with_holdings(account_id, holding_repo)

    holding_repo.delete_by_account.assert_called_once_with(account_id)
    client.table.assert_called_once_with("accounts")
    client.table.return_value.delete.assert_called_once()
    client.table.return_value.delete.return_value.eq.assert_called_once_with(
        "id", str(account_id)
    )
