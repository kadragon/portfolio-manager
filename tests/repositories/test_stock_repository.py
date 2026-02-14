"""Tests for stock repository."""

from datetime import datetime
from unittest.mock import MagicMock, Mock
from uuid import uuid4

import pytest

from portfolio_manager.repositories.stock_repository import StockRepository


def test_create_stock_returns_stock_with_id():
    """Should create a stock and return it with an ID."""
    # Arrange
    mock_client = Mock()
    mock_response = MagicMock()
    group_id = uuid4()
    stock_id = uuid4()

    mock_response.data = [
        {
            "id": str(stock_id),
            "ticker": "AAPL",
            "group_id": str(group_id),
            "created_at": datetime.now().isoformat(),
            "updated_at": datetime.now().isoformat(),
        }
    ]
    mock_client.table.return_value.insert.return_value.execute.return_value = (
        mock_response
    )

    repository = StockRepository(mock_client)

    # Act
    stock = repository.create("AAPL", group_id)

    # Assert
    assert stock is not None
    assert stock.ticker == "AAPL"
    assert stock.group_id == group_id
    assert stock.id is not None
    mock_client.table.assert_called_once_with("stocks")


def test_list_by_group_returns_stocks_for_group():
    """Should return all stocks for a given group."""
    # Arrange
    mock_client = Mock()
    mock_response = MagicMock()
    group_id = uuid4()
    now = datetime.now().isoformat()

    mock_response.data = [
        {
            "id": str(uuid4()),
            "ticker": "AAPL",
            "group_id": str(group_id),
            "created_at": now,
            "updated_at": now,
        },
        {
            "id": str(uuid4()),
            "ticker": "GOOGL",
            "group_id": str(group_id),
            "created_at": now,
            "updated_at": now,
        },
    ]
    mock_client.table.return_value.select.return_value.eq.return_value.execute.return_value = mock_response

    repository = StockRepository(mock_client)

    # Act
    stocks = repository.list_by_group(group_id)

    # Assert
    assert len(stocks) == 2
    assert stocks[0].ticker == "AAPL"
    assert stocks[1].ticker == "GOOGL"
    assert all(s.group_id == group_id for s in stocks)
    mock_client.table.assert_called_once_with("stocks")


def test_delete_removes_stock():
    """Should delete a stock by ID."""
    # Arrange
    mock_client = Mock()
    mock_response = MagicMock()
    stock_id = uuid4()

    mock_response.data = []
    mock_client.table.return_value.delete.return_value.eq.return_value.execute.return_value = mock_response

    repository = StockRepository(mock_client)

    # Act
    repository.delete(stock_id)

    # Assert
    mock_client.table.assert_called_once_with("stocks")
    mock_client.table.return_value.delete.assert_called_once()
    mock_client.table.return_value.delete.return_value.eq.assert_called_once_with(
        "id", str(stock_id)
    )


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


def test_stock_repository_updates_exchange():
    """Should update a stock exchange by ID."""
    stock_id = uuid4()
    response = Mock()
    response.data = [
        {
            "id": str(stock_id),
            "ticker": "SCHD",
            "group_id": str(uuid4()),
            "exchange": "NAS",
            "created_at": "2026-01-03T00:00:00",
            "updated_at": "2026-01-03T00:00:00",
        }
    ]

    client = Mock()
    client.table.return_value.update.return_value.eq.return_value.execute.return_value = response

    repository = StockRepository(client)
    stock = repository.update_exchange(stock_id, "NAS")

    client.table.assert_called_once_with("stocks")
    client.table.return_value.update.assert_called_once_with({"exchange": "NAS"})
    client.table.return_value.update.return_value.eq.assert_called_once_with(
        "id", str(stock_id)
    )
    assert stock is not None
    assert stock.exchange == "NAS"


def test_create_stock_raises_when_no_rows_returned():
    """Should raise ValueError when create returns empty data."""
    group_id = uuid4()
    response = Mock()
    response.data = []
    client = Mock()
    client.table.return_value.insert.return_value.execute.return_value = response
    repository = StockRepository(client)

    with pytest.raises(ValueError, match="Failed to create stock"):
        repository.create("AAPL", group_id)


def test_list_by_group_returns_empty_when_no_rows():
    """Should return empty list when group has no stocks."""
    group_id = uuid4()
    response = Mock()
    response.data = []
    client = Mock()
    client.table.return_value.select.return_value.eq.return_value.execute.return_value = response
    repository = StockRepository(client)

    assert repository.list_by_group(group_id) == []


def test_list_all_returns_empty_when_no_rows():
    """Should return empty list when there are no stocks."""
    response = Mock()
    response.data = []
    client = Mock()
    client.table.return_value.select.return_value.execute.return_value = response
    repository = StockRepository(client)

    assert repository.list_all() == []


def test_stock_repository_update_returns_updated_stock():
    """Should update ticker and return updated stock."""
    stock_id = uuid4()
    group_id = uuid4()
    response = Mock()
    response.data = [
        {
            "id": str(stock_id),
            "ticker": "MSFT",
            "group_id": str(group_id),
            "created_at": "2026-01-03T00:00:00",
            "updated_at": "2026-01-03T00:00:00",
        }
    ]
    client = Mock()
    client.table.return_value.update.return_value.eq.return_value.execute.return_value = response
    repository = StockRepository(client)

    stock = repository.update(stock_id, "MSFT")

    assert stock.ticker == "MSFT"
    client.table.return_value.update.assert_called_once_with({"ticker": "MSFT"})


def test_stock_repository_update_raises_when_no_rows():
    """Should raise ValueError when ticker update returns no rows."""
    stock_id = uuid4()
    response = Mock()
    response.data = []
    client = Mock()
    client.table.return_value.update.return_value.eq.return_value.execute.return_value = response
    repository = StockRepository(client)

    with pytest.raises(ValueError, match="Failed to update stock"):
        repository.update(stock_id, "MSFT")


def test_stock_repository_get_by_id_returns_none_when_not_found():
    """Should return None if stock id does not exist."""
    stock_id = uuid4()
    response = Mock()
    response.data = []
    client = Mock()
    client.table.return_value.select.return_value.eq.return_value.execute.return_value = response
    repository = StockRepository(client)

    assert repository.get_by_id(stock_id) is None


def test_stock_repository_get_by_ticker_returns_none_when_not_found():
    """Should return None if ticker does not exist."""
    response = Mock()
    response.data = []
    client = Mock()
    client.table.return_value.select.return_value.eq.return_value.execute.return_value = response
    repository = StockRepository(client)

    assert repository.get_by_ticker("UNKNOWN") is None


def test_stock_repository_update_group_returns_updated_stock():
    """Should move stock to target group and return updated stock."""
    stock_id = uuid4()
    group_id = uuid4()
    response = Mock()
    response.data = [
        {
            "id": str(stock_id),
            "ticker": "AAPL",
            "group_id": str(group_id),
            "created_at": "2026-01-03T00:00:00",
            "updated_at": "2026-01-03T00:00:00",
        }
    ]
    client = Mock()
    client.table.return_value.update.return_value.eq.return_value.execute.return_value = response
    repository = StockRepository(client)

    moved = repository.update_group(stock_id, group_id)

    assert moved.group_id == group_id
    client.table.return_value.update.assert_called_once_with(
        {"group_id": str(group_id)}
    )


def test_stock_repository_update_group_raises_when_no_rows():
    """Should raise ValueError when move operation returns no rows."""
    stock_id = uuid4()
    group_id = uuid4()
    response = Mock()
    response.data = []
    client = Mock()
    client.table.return_value.update.return_value.eq.return_value.execute.return_value = response
    repository = StockRepository(client)

    with pytest.raises(ValueError, match="Failed to move stock"):
        repository.update_group(stock_id, group_id)


def test_stock_repository_update_exchange_raises_when_no_rows():
    """Should raise ValueError when exchange update returns no rows."""
    stock_id = uuid4()
    response = Mock()
    response.data = []
    client = Mock()
    client.table.return_value.update.return_value.eq.return_value.execute.return_value = response
    repository = StockRepository(client)

    with pytest.raises(ValueError, match="Failed to update stock exchange"):
        repository.update_exchange(stock_id, "NYS")
