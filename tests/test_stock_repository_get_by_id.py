"""Tests for stock repository get by id."""

from uuid import uuid4
from unittest.mock import Mock

from portfolio_manager.repositories.stock_repository import StockRepository


def test_stock_repository_gets_stock_by_id():
    """Should fetch a stock by id."""
    stock_id = uuid4()
    response = Mock()
    response.data = [
        {
            "id": str(stock_id),
            "ticker": "AAPL",
            "group_id": str(uuid4()),
            "created_at": "2026-01-03T00:00:00",
            "updated_at": "2026-01-03T00:00:00",
        }
    ]

    client = Mock()
    client.table.return_value.select.return_value.eq.return_value.execute.return_value = response

    repository = StockRepository(client)
    stock = repository.get_by_id(stock_id)

    client.table.assert_called_once_with("stocks")
    client.table.return_value.select.assert_called_once_with("*")
    client.table.return_value.select.return_value.eq.assert_called_once_with(
        "id", str(stock_id)
    )
    assert stock is not None
    assert stock.ticker == "AAPL"


def test_stock_repository_gets_stock_by_ticker():
    """Should fetch a stock by ticker."""
    stock_id = uuid4()
    response = Mock()
    response.data = [
        {
            "id": str(stock_id),
            "ticker": "310970",
            "group_id": str(uuid4()),
            "created_at": "2026-01-03T00:00:00",
            "updated_at": "2026-01-03T00:00:00",
        }
    ]

    client = Mock()
    client.table.return_value.select.return_value.eq.return_value.execute.return_value = response

    repository = StockRepository(client)
    stock = repository.get_by_ticker("310970")

    client.table.assert_called_once_with("stocks")
    client.table.return_value.select.assert_called_once_with("*")
    client.table.return_value.select.return_value.eq.assert_called_once_with(
        "ticker", "310970"
    )
    assert stock is not None
    assert stock.ticker == "310970"
