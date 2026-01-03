"""Tests for stock repository."""

from datetime import datetime
from unittest.mock import MagicMock, Mock
from uuid import uuid4

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
