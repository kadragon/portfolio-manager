"""Tests for deposit repository."""

from decimal import Decimal
from datetime import date

from portfolio_manager.models.deposit import Deposit
from portfolio_manager.repositories.deposit_repository import DepositRepository


def test_deposit_repository_creates_deposit():
    repo = DepositRepository()
    deposit = repo.create(
        amount=Decimal("1000000"),
        deposit_date=date(2026, 1, 4),
        note="Initial funding",
    )

    assert isinstance(deposit, Deposit)
    assert deposit.amount == Decimal("1000000")
    assert deposit.deposit_date == date(2026, 1, 4)
    assert deposit.note == "Initial funding"


def test_deposit_repository_updates_deposit():
    repo = DepositRepository()
    deposit = repo.create(amount=Decimal("1000000"), deposit_date=date(2026, 1, 4))

    updated = repo.update(deposit.id, amount=Decimal("2000000"), note="Updated funding")

    assert updated.amount == Decimal("2000000")
    assert updated.note == "Updated funding"


def test_deposit_repository_updates_note_to_null_when_explicit_none():
    repo = DepositRepository()
    deposit = repo.create(
        amount=Decimal("1000000"), deposit_date=date(2026, 1, 4), note="Initial"
    )

    updated = repo.update(deposit.id, amount=Decimal("2000000"), note=None)

    assert updated.note is None


def test_deposit_repository_omits_note_when_not_provided():
    repo = DepositRepository()
    deposit = repo.create(
        amount=Decimal("1000000"), deposit_date=date(2026, 1, 4), note="Existing"
    )

    updated = repo.update(deposit.id, amount=Decimal("2000000"))

    assert updated.note == "Existing"


def test_deposit_repository_lists_all():
    repo = DepositRepository()
    repo.create(amount=Decimal("1000000"), deposit_date=date(2026, 1, 4))

    deposits = repo.list_all()

    assert len(deposits) == 1


def test_deposit_repository_gets_by_date():
    repo = DepositRepository()
    repo.create(amount=Decimal("1000000"), deposit_date=date(2026, 1, 4))

    deposit = repo.get_by_date(date(2026, 1, 4))

    assert deposit is not None
    assert deposit.amount == Decimal("1000000")


def test_deposit_repository_get_by_date_returns_none():
    repo = DepositRepository()
    assert repo.get_by_date(date(2099, 1, 1)) is None


def test_deposit_repository_deletes_deposit():
    repo = DepositRepository()
    deposit = repo.create(amount=Decimal("1000000"), deposit_date=date(2026, 1, 4))

    repo.delete(deposit.id)

    assert repo.list_all() == []


def test_deposit_repository_gets_total():
    repo = DepositRepository()
    repo.create(amount=Decimal("1000000"), deposit_date=date(2026, 1, 4))
    repo.create(amount=Decimal("500000"), deposit_date=date(2026, 2, 4))

    total = repo.get_total()

    assert total == Decimal("1500000")


def test_deposit_repository_gets_first_deposit_date():
    repo = DepositRepository()
    repo.create(amount=Decimal("500000"), deposit_date=date(2026, 2, 4))
    repo.create(amount=Decimal("1000000"), deposit_date=date(2026, 1, 4))

    first = repo.get_first_deposit_date()

    assert first == date(2026, 1, 4)


def test_deposit_repository_gets_first_deposit_date_returns_none():
    repo = DepositRepository()
    assert repo.get_first_deposit_date() is None
