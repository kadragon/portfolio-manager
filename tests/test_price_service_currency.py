"""Test price service returns currency information."""

from decimal import Decimal
from unittest.mock import Mock

from portfolio_manager.services.kis_price_parser import PriceQuote
from portfolio_manager.services.price_service import PriceService


def test_get_stock_price_returns_currency():
    """주식 가격 조회 시 화폐 단위도 함께 반환한다."""
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


def test_get_stock_price_returns_krw_for_domestic():
    """국내 주식은 KRW를 반환한다."""
    # Given: Mock price client for domestic stock
    price_client = Mock()
    price_client.get_price.return_value = PriceQuote(
        symbol="005930", name="삼성전자", price=70000, market="KR", currency="KRW"
    )

    service = PriceService(price_client)

    # When: 국내 주식 가격 조회
    price, currency = service.get_stock_price("005930")

    # Then: KRW가 반환됨
    assert price == Decimal("70000")
    assert currency == "KRW"
