"""Test price fetching service."""

from decimal import Decimal
from unittest.mock import Mock

from portfolio_manager.services.kis_price_parser import PriceQuote
from portfolio_manager.services.price_service import PriceService


def test_get_stock_price_returns_price():
    """주식의 현재가를 조회한다."""
    # Given: Mock price client
    price_client = Mock()
    price_client.get_price.return_value = PriceQuote(
        symbol="AAPL", name="Apple Inc.", price=150.0, market="US", currency="USD"
    )

    service = PriceService(price_client)

    # When: 주식 가격 조회
    price, currency = service.get_stock_price("AAPL")

    # Then: 가격과 화폐 단위가 반환됨
    assert price == Decimal("150.0")
    assert currency == "USD"
    price_client.get_price.assert_called_once_with("AAPL")
