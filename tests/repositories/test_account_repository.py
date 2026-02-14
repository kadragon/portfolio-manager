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


def test_account_repository_create_raises_when_no_rows_returned():
    """Should raise ValueError when create returns empty response."""
    response = Mock()
    response.data = []
    client = Mock()
    client.table.return_value.insert.return_value.execute.return_value = response

    repository = AccountRepository(client)

    try:
        repository.create(name="Main Account", cash_balance=Decimal("100.0"))
        assert False, "Expected ValueError"
    except ValueError as exc:
        assert "Failed to create account" in str(exc)


def test_account_repository_list_all_returns_empty_when_no_data():
    """Should return empty list when accounts table returns no rows."""
    response = Mock()
    response.data = []
    client = Mock()
    client.table.return_value.select.return_value.execute.return_value = response

    repository = AccountRepository(client)

    assert repository.list_all() == []


def test_account_repository_updates_account():
    """Should update account fields and return updated account."""
    account_id = uuid4()
    response = Mock()
    response.data = [
        {
            "id": str(account_id),
            "name": "Updated",
            "cash_balance": "200.50",
            "created_at": "2026-01-03T00:00:00",
            "updated_at": "2026-01-03T00:00:00",
        }
    ]
    client = Mock()
    client.table.return_value.update.return_value.eq.return_value.execute.return_value = response

    repository = AccountRepository(client)
    account = repository.update(
        account_id, name="Updated", cash_balance=Decimal("200.50")
    )

    assert account.name == "Updated"
    assert account.cash_balance == Decimal("200.50")
    client.table.return_value.update.assert_called_once_with(
        {"name": "Updated", "cash_balance": "200.50"}
    )


def test_account_repository_update_raises_when_no_rows_returned():
    """Should raise ValueError when update returns no rows."""
    account_id = uuid4()
    response = Mock()
    response.data = []
    client = Mock()
    client.table.return_value.update.return_value.eq.return_value.execute.return_value = response

    repository = AccountRepository(client)

    try:
        repository.update(account_id, name="Updated", cash_balance=Decimal("10"))
        assert False, "Expected ValueError"
    except ValueError as exc:
        assert "Failed to update account" in str(exc)
