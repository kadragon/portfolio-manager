"""Tests for account repository."""

from decimal import Decimal

from portfolio_manager.repositories.account_repository import AccountRepository
from portfolio_manager.repositories.holding_repository import HoldingRepository


def test_account_repository_creates_account_with_cash_balance():
    repo = AccountRepository()
    account = repo.create(name="Main Account", cash_balance=Decimal("100000.50"))

    assert account.name == "Main Account"
    assert account.cash_balance == Decimal("100000.50")
    assert account.id is not None


def test_account_repository_list_all():
    repo = AccountRepository()
    repo.create(name="Account 1", cash_balance=Decimal("100"))
    repo.create(name="Account 2", cash_balance=Decimal("200"))

    accounts = repo.list_all()
    assert len(accounts) == 2


def test_account_repository_list_all_returns_empty_when_no_data():
    repo = AccountRepository()
    assert repo.list_all() == []


def test_account_repository_updates_account():
    repo = AccountRepository()
    account = repo.create(name="Original", cash_balance=Decimal("100"))

    updated = repo.update(account.id, name="Updated", cash_balance=Decimal("200.50"))

    assert updated.name == "Updated"
    assert updated.cash_balance == Decimal("200.50")


def test_account_repository_deletes_holdings_before_account():
    repo = AccountRepository()
    holding_repo = HoldingRepository()
    account = repo.create(name="Test", cash_balance=Decimal("0"))

    repo.delete_with_holdings(account.id, holding_repo)

    assert repo.list_all() == []
