"""Tests for stock repository."""

import pytest

from portfolio_manager.repositories.group_repository import GroupRepository
from portfolio_manager.repositories.stock_repository import StockRepository


@pytest.fixture
def group_id():
    repo = GroupRepository()
    group = repo.create("Test Group")
    return group.id


def test_create_stock_returns_stock_with_id(group_id):
    repo = StockRepository()
    stock = repo.create("AAPL", group_id)

    assert stock is not None
    assert stock.ticker == "AAPL"
    assert stock.group_id == group_id
    assert stock.id is not None


def test_list_by_group_returns_stocks_for_group(group_id):
    repo = StockRepository()
    repo.create("AAPL", group_id)
    repo.create("GOOGL", group_id)

    stocks = repo.list_by_group(group_id)

    assert len(stocks) == 2
    tickers = {s.ticker for s in stocks}
    assert tickers == {"AAPL", "GOOGL"}


def test_list_by_group_returns_empty_when_no_rows(group_id):
    repo = StockRepository()
    assert repo.list_by_group(group_id) == []


def test_list_all(group_id):
    repo = StockRepository()
    repo.create("AAPL", group_id)
    assert len(repo.list_all()) == 1


def test_list_all_returns_empty_when_no_rows():
    repo = StockRepository()
    assert repo.list_all() == []


def test_delete_removes_stock(group_id):
    repo = StockRepository()
    stock = repo.create("AAPL", group_id)

    repo.delete(stock.id)

    assert repo.list_all() == []


def test_stock_repository_gets_stock_by_id(group_id):
    repo = StockRepository()
    stock = repo.create("AAPL", group_id)

    found = repo.get_by_id(stock.id)

    assert found is not None
    assert found.ticker == "AAPL"


def test_stock_repository_get_by_id_returns_none_when_not_found():
    from uuid import uuid4

    repo = StockRepository()
    assert repo.get_by_id(uuid4()) is None


def test_stock_repository_gets_stock_by_ticker(group_id):
    repo = StockRepository()
    repo.create("310970", group_id)

    found = repo.get_by_ticker("310970")

    assert found is not None
    assert found.ticker == "310970"


def test_stock_repository_get_by_ticker_returns_none_when_not_found():
    repo = StockRepository()
    assert repo.get_by_ticker("UNKNOWN") is None


def test_stock_repository_update_returns_updated_stock(group_id):
    repo = StockRepository()
    stock = repo.create("AAPL", group_id)

    updated = repo.update(stock.id, "MSFT")

    assert updated.ticker == "MSFT"


def test_stock_repository_updates_exchange(group_id):
    repo = StockRepository()
    stock = repo.create("SCHD", group_id)

    updated = repo.update_exchange(stock.id, "NAS")

    assert updated.exchange == "NAS"


def test_stock_repository_update_group(group_id):
    group_repo = GroupRepository()
    new_group = group_repo.create("New Group")

    repo = StockRepository()
    stock = repo.create("AAPL", group_id)

    moved = repo.update_group(stock.id, new_group.id)

    assert moved.group_id == new_group.id
