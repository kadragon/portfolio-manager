"""Tests for account deletion with holdings cascade."""

from uuid import uuid4
from unittest.mock import MagicMock, Mock

from portfolio_manager.repositories.account_repository import AccountRepository


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
