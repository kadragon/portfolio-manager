"""Test price fetching service."""

from decimal import Decimal
from unittest.mock import Mock
from datetime import date

from portfolio_manager.services.kis.kis_price_parser import PriceQuote
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
    price, currency, name = service.get_stock_price("AAPL")

    # Then: 가격과 화폐 단위가 반환됨
    assert price == Decimal("150.0")
    assert currency == "USD"
    assert name == "Apple Inc."
    price_client.get_price.assert_called_once_with("AAPL")


def test_get_stock_price_returns_currency():
    """주식 가격 조회 시 화폐 단위도 함께 반환한다."""
    # Given: Mock price client
    price_client = Mock()
    price_client.get_price.return_value = PriceQuote(
        symbol="AAPL", name="Apple Inc.", price=150.0, market="US", currency="USD"
    )

    service = PriceService(price_client)

    # When: 주식 가격 조회
    price, currency, name = service.get_stock_price("AAPL")

    # Then: 가격과 화폐 단위가 반환됨
    assert price == Decimal("150.0")
    assert currency == "USD"
    assert name == "Apple Inc."
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
    price, currency, name = service.get_stock_price("005930")

    # Then: KRW가 반환됨
    assert price == Decimal("70000")
    assert currency == "KRW"
    assert name == "삼성전자"


def test_price_quote_has_currency_field():
    """PriceQuote는 currency 필드를 가진다."""
    quote = PriceQuote(
        symbol="005930",
        name="삼성전자",
        price=70000,
        market="KR",
        currency="KRW",
    )

    assert quote.currency == "KRW"


def test_domestic_price_uses_krw():
    """국내 주식은 KRW 화폐를 사용한다."""
    quote = PriceQuote(
        symbol="005930",
        name="삼성전자",
        price=70000,
        market="KR",
        currency="KRW",
    )

    assert quote.currency == "KRW"


def test_overseas_price_uses_usd():
    """해외 주식은 USD 화폐를 사용한다."""
    quote = PriceQuote(
        symbol="AAPL",
        name="Apple Inc.",
        price=150.0,
        market="US",
        currency="USD",
    )

    assert quote.currency == "USD"


def test_get_stock_change_rates_returns_percentages():
    """현재가 대비 1Y/6M/1M 변동률을 계산한다."""
    price_client = Mock()
    price_client.get_price.return_value = PriceQuote(
        symbol="AAPL", name="Apple Inc.", price=120.0, market="US", currency="USD"
    )
    price_client.get_historical_close.side_effect = lambda ticker, target_date: {
        date(2024, 1, 15): 100.0,
        date(2024, 7, 15): 80.0,
        date(2024, 12, 13): 60.0,
    }[target_date]

    service = PriceService(price_client)

    change_rates = service.get_stock_change_rates("AAPL", as_of=date(2025, 1, 15))

    assert change_rates["1y"] == Decimal("20")
    assert change_rates["6m"] == Decimal("50")
    assert change_rates["1m"] == Decimal("100")


def test_get_stock_change_rates_adjusts_to_previous_business_day():
    """휴장일이면 이전 영업일로 보정해 과거 종가를 조회한다."""
    price_client = Mock()
    price_client.get_price.return_value = PriceQuote(
        symbol="AAPL", name="Apple Inc.", price=120.0, market="US", currency="USD"
    )
    price_client.get_historical_close.side_effect = lambda ticker, target_date: {
        date(2024, 1, 12): 100.0,  # 2024-01-13 (Sat) -> 2024-01-12 (Fri)
        date(2024, 7, 12): 100.0,
        date(2024, 12, 13): 100.0,
    }[target_date]

    service = PriceService(price_client)

    change_rates = service.get_stock_change_rates("AAPL", as_of=date(2025, 1, 13))

    assert change_rates["1y"] == Decimal("20")
    price_client.get_historical_close.assert_any_call("AAPL", date(2024, 1, 12))
