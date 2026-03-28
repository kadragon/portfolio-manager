"""Tests for DepositRepository.get_first_deposit_date."""

from datetime import date
from decimal import Decimal

import pytest

from portfolio_manager.repositories.deposit_repository import DepositRepository
from portfolio_manager.services.database import db, ALL_MODELS


@pytest.fixture(autouse=True)
def setup_test_db():
    db.init(":memory:", pragmas={"foreign_keys": 1})
    db.create_tables(ALL_MODELS)
    yield
    db.drop_tables(ALL_MODELS)
    db.close()


def test_get_first_deposit_date_returns_earliest():
    repo = DepositRepository()
    repo.create(amount=Decimal("1000000"), deposit_date=date(2024, 3, 15))
    repo.create(amount=Decimal("500000"), deposit_date=date(2024, 1, 15))

    result = repo.get_first_deposit_date()

    assert result == date(2024, 1, 15)


def test_get_first_deposit_date_returns_none_when_empty():
    repo = DepositRepository()
    result = repo.get_first_deposit_date()

    assert result is None
