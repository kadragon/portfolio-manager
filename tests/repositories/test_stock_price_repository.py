"""Tests for stock price repository."""

from datetime import date
from decimal import Decimal
from uuid import uuid4
from unittest.mock import Mock

from portfolio_manager.models.stock_price import StockPrice
from portfolio_manager.repositories.stock_price_repository import StockPriceRepository


def test_stock_price_repository_gets_by_ticker_and_date():
    """Should fetch cached price by ticker and date."""
    price_id = uuid4()
    response = Mock()
    response.data = [
        {
            "id": str(price_id),
            "ticker": "QQQ",
            "price": "410.12",
            "currency": "USD",
            "name": "Invesco QQQ Trust",
            "exchange": "NAS",
            "price_date": "2026-01-04",
            "created_at": "2026-01-04T00:00:00",
            "updated_at": "2026-01-04T00:00:00",
        }
    ]

    client = Mock()
    client.table.return_value.select.return_value.eq.return_value.eq.return_value.execute.return_value = response

    repository = StockPriceRepository(client)

    result = repository.get_by_ticker_and_date("QQQ", date(2026, 1, 4))

    client.table.assert_called_once_with("stock_prices")
    client.table.return_value.select.assert_called_once_with("*")
    client.table.return_value.select.return_value.eq.assert_any_call("ticker", "QQQ")
    client.table.return_value.select.return_value.eq.return_value.eq.assert_called_once_with(
        "price_date", "2026-01-04"
    )

    assert isinstance(result, StockPrice)
    assert result is not None
    assert result.id == price_id
    assert result.price == Decimal("410.12")
