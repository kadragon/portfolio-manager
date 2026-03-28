"""Tests for holding repository."""

from decimal import Decimal

import pytest

from portfolio_manager.repositories.account_repository import AccountRepository
from portfolio_manager.repositories.group_repository import GroupRepository
from portfolio_manager.repositories.holding_repository import HoldingRepository
from portfolio_manager.repositories.stock_repository import StockRepository


@pytest.fixture
def account_id():
    return AccountRepository().create("Test Account", Decimal("0")).id


@pytest.fixture
def stock_id():
    group = GroupRepository().create("Test Group")
    return StockRepository().create("AAPL", group.id).id


def test_holding_repository_creates_holding_with_decimal_quantity(account_id, stock_id):
    repo = HoldingRepository()
    holding = repo.create(
        account_id=account_id, stock_id=stock_id, quantity=Decimal("10.75")
    )

    assert holding.quantity == Decimal("10.75")
    assert holding.account_id == account_id
    assert holding.stock_id == stock_id


def test_holding_repository_list_by_account(account_id, stock_id):
    repo = HoldingRepository()
    repo.create(account_id=account_id, stock_id=stock_id, quantity=Decimal("3.1415"))

    holdings = repo.list_by_account(account_id)

    assert len(holdings) == 1
    assert holdings[0].quantity == Decimal("3.1415")


def test_holding_repository_list_by_account_returns_empty_when_no_rows(account_id):
    repo = HoldingRepository()
    assert repo.list_by_account(account_id) == []


def test_holding_repository_deletes_by_id(account_id, stock_id):
    repo = HoldingRepository()
    holding = repo.create(
        account_id=account_id, stock_id=stock_id, quantity=Decimal("1")
    )

    repo.delete(holding.id)

    assert repo.list_by_account(account_id) == []


def test_holding_repository_deletes_by_account(account_id, stock_id):
    repo = HoldingRepository()
    repo.create(account_id=account_id, stock_id=stock_id, quantity=Decimal("1"))

    repo.delete_by_account(account_id)

    assert repo.list_by_account(account_id) == []


def test_holding_repository_update_returns_updated_holding(account_id, stock_id):
    repo = HoldingRepository()
    holding = repo.create(
        account_id=account_id, stock_id=stock_id, quantity=Decimal("5")
    )

    updated = repo.update(holding.id, quantity=Decimal("9.5"))

    assert updated.quantity == Decimal("9.5")


def test_aggregate_holdings_by_stock(account_id, stock_id):
    repo = HoldingRepository()
    repo.create(account_id=account_id, stock_id=stock_id, quantity=Decimal("10"))

    # Create second account with same stock
    account2 = AccountRepository().create("Account 2", Decimal("0"))
    repo.create(account_id=account2.id, stock_id=stock_id, quantity=Decimal("5"))

    aggregated = repo.get_aggregated_holdings_by_stock()

    assert stock_id in aggregated
    assert aggregated[stock_id] == Decimal("15")


def test_aggregate_holdings_by_stock_empty():
    repo = HoldingRepository()
    assert repo.get_aggregated_holdings_by_stock() == {}


def test_bulk_update_by_account_updates_holdings(account_id, stock_id):
    repo = HoldingRepository()
    holding = repo.create(
        account_id=account_id, stock_id=stock_id, quantity=Decimal("5")
    )

    # Create second stock and holding
    group = GroupRepository().create("Group 2")
    stock2 = StockRepository().create("GOOGL", group.id)
    holding2 = repo.create(
        account_id=account_id, stock_id=stock2.id, quantity=Decimal("3")
    )

    updated = repo.bulk_update_by_account(
        account_id,
        [
            (holding.id, Decimal("11.5")),
            (holding2.id, Decimal("7")),
        ],
    )

    assert len(updated) == 2
    assert updated[0].quantity == Decimal("11.5")
    assert updated[1].quantity == Decimal("7")


def test_bulk_update_by_account_returns_empty_for_no_updates(account_id):
    repo = HoldingRepository()
    assert repo.bulk_update_by_account(account_id, []) == []


def test_bulk_update_by_account_raises_on_duplicate_ids(account_id, stock_id):
    repo = HoldingRepository()
    holding = repo.create(
        account_id=account_id, stock_id=stock_id, quantity=Decimal("5")
    )

    with pytest.raises(ValueError, match="duplicate holding_ids"):
        repo.bulk_update_by_account(
            account_id,
            [(holding.id, Decimal("1")), (holding.id, Decimal("2"))],
        )


def test_bulk_update_by_account_raises_on_non_positive_quantity(account_id, stock_id):
    repo = HoldingRepository()
    holding = repo.create(
        account_id=account_id, stock_id=stock_id, quantity=Decimal("5")
    )

    with pytest.raises(ValueError, match="quantity must be greater than zero"):
        repo.bulk_update_by_account(
            account_id,
            [(holding.id, Decimal("0"))],
        )


def test_bulk_update_by_account_raises_when_holding_not_in_account(
    account_id, stock_id
):
    repo = HoldingRepository()
    holding = repo.create(
        account_id=account_id, stock_id=stock_id, quantity=Decimal("5")
    )

    other_account = AccountRepository().create("Other", Decimal("0"))

    with pytest.raises(ValueError, match="all holdings must belong to account"):
        repo.bulk_update_by_account(
            other_account.id,
            [(holding.id, Decimal("1"))],
        )
