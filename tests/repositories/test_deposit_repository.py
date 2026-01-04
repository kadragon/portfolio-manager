"""Tests for deposit repository."""

from decimal import Decimal
from datetime import date
from uuid import uuid4
from unittest.mock import Mock

from portfolio_manager.models.deposit import Deposit
from portfolio_manager.repositories.deposit_repository import DepositRepository


def test_deposit_repository_creates_deposit():
    """Should create a deposit."""
    account_id = uuid4()
    deposit_id = uuid4()
    deposit_date = date(2026, 1, 4)

    response = Mock()
    response.data = [
        {
            "id": str(deposit_id),
            "account_id": str(account_id),
            "amount": "1000000",
            "deposit_date": "2026-01-04",
            "note": "Initial funding",
            "created_at": "2026-01-04T00:00:00",
            "updated_at": "2026-01-04T00:00:00",
        }
    ]

    client = Mock()
    client.table.return_value.insert.return_value.execute.return_value = response

    repository = DepositRepository(client)
    deposit = repository.create(
        account_id=account_id,
        amount=Decimal("1000000"),
        deposit_date=deposit_date,
        note="Initial funding",
    )

    client.table.assert_called_once_with("deposits")
    client.table.return_value.insert.assert_called_once_with(
        {
            "account_id": str(account_id),
            "amount": "1000000",
            "deposit_date": "2026-01-04",
            "note": "Initial funding",
        }
    )

    assert isinstance(deposit, Deposit)
    assert deposit.id == deposit_id
    assert deposit.account_id == account_id
    assert deposit.amount == Decimal("1000000")
    assert deposit.deposit_date == deposit_date
    assert deposit.note == "Initial funding"


def test_deposit_repository_lists_by_account():
    """Should list deposits by account."""
    account_id = uuid4()
    deposit_id = uuid4()

    response = Mock()
    response.data = [
        {
            "id": str(deposit_id),
            "account_id": str(account_id),
            "amount": "1000000",
            "deposit_date": "2026-01-04",
            "note": "Initial funding",
            "created_at": "2026-01-04T00:00:00",
            "updated_at": "2026-01-04T00:00:00",
        }
    ]

    client = Mock()
    client.table.return_value.select.return_value.eq.return_value.order.return_value.execute.return_value = response

    repository = DepositRepository(client)
    deposits = repository.list_by_account(account_id)

    client.table.assert_called_once_with("deposits")
    client.table.return_value.select.assert_called_once_with("*")
    client.table.return_value.select.return_value.eq.assert_called_once_with(
        "account_id", str(account_id)
    )

    assert len(deposits) == 1
    assert deposits[0].id == deposit_id
    assert deposits[0].amount == Decimal("1000000")


def test_deposit_repository_deletes_deposit():
    """Should delete a deposit."""
    deposit_id = uuid4()

    client = Mock()
    client.table.return_value.delete.return_value.eq.return_value.execute.return_value = Mock()

    repository = DepositRepository(client)
    repository.delete(deposit_id)

    client.table.assert_called_once_with("deposits")
    client.table.return_value.delete.assert_called_once()
    client.table.return_value.delete.return_value.eq.assert_called_once_with(
        "id", str(deposit_id)
    )


def test_deposit_repository_gets_total_by_account():
    """Should get total deposit amount by account."""
    account_id = uuid4()

    # Mock response for list_by_account which is used internally or we can use sum query if supported
    # Assuming implementation fetches all and sums them up for now, or uses RPC if optimized.
    # Let's assume we implement it by fetching all deposits first (simplest).

    response = Mock()
    response.data = [
        {
            "id": str(uuid4()),
            "account_id": str(account_id),
            "amount": "1000000",
            "deposit_date": "2026-01-04",
            "note": "Initial",
            "created_at": "2026-01-04T00:00:00",
            "updated_at": "2026-01-04T00:00:00",
        },
        {
            "id": str(uuid4()),
            "account_id": str(account_id),
            "amount": "500000",
            "deposit_date": "2026-02-04",
            "note": "Second",
            "created_at": "2026-02-04T00:00:00",
            "updated_at": "2026-02-04T00:00:00",
        },
    ]

    client = Mock()
    client.table.return_value.select.return_value.eq.return_value.order.return_value.execute.return_value = response

    repository = DepositRepository(client)
    total = repository.get_total_by_account(account_id)

    assert total == Decimal("1500000")
