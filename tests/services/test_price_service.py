"""Test price fetching service."""

from decimal import Decimal
from unittest.mock import Mock
from datetime import date
from uuid import uuid4

import httpx

from portfolio_manager.models.stock_price import StockPrice
from portfolio_manager.services.kis.kis_price_parser import PriceQuote
from portfolio_manager.services.kis.kis_unified_price_client import (
    KisUnifiedPriceClient,
)
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
    price, currency, name, exchange = service.get_stock_price("AAPL")

    # Then: 가격과 화폐 단위가 반환됨
    assert price == Decimal("150.0")
    assert currency == "USD"
    assert name == "Apple Inc."
    assert exchange is None
    price_client.get_price.assert_called_once_with("AAPL", preferred_exchange=None)


def test_get_stock_price_returns_currency():
    """주식 가격 조회 시 화폐 단위도 함께 반환한다."""
    # Given: Mock price client
    price_client = Mock()
    price_client.get_price.return_value = PriceQuote(
        symbol="AAPL", name="Apple Inc.", price=150.0, market="US", currency="USD"
    )

    service = PriceService(price_client)

    # When: 주식 가격 조회
    price, currency, name, exchange = service.get_stock_price("AAPL")

    # Then: 가격과 화폐 단위가 반환됨
    assert price == Decimal("150.0")
    assert currency == "USD"
    assert name == "Apple Inc."
    assert exchange is None
    price_client.get_price.assert_called_once_with("AAPL", preferred_exchange=None)


def test_get_stock_price_returns_krw_for_domestic():
    """국내 주식은 KRW를 반환한다."""
    # Given: Mock price client for domestic stock
    price_client = Mock()
    price_client.get_price.return_value = PriceQuote(
        symbol="005930", name="삼성전자", price=70000, market="KR", currency="KRW"
    )

    service = PriceService(price_client)

    # When: 국내 주식 가격 조회
    price, currency, name, exchange = service.get_stock_price("005930")

    # Then: KRW가 반환됨
    assert price == Decimal("70000")
    assert currency == "KRW"
    assert name == "삼성전자"
    assert exchange is None


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
    price_client.get_historical_close.side_effect = (
        lambda ticker, target_date, preferred_exchange=None: {
            date(2024, 1, 15): 100.0,
            date(2024, 7, 15): 80.0,
            date(2024, 12, 13): 60.0,
        }[target_date]
    )

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
    price_client.get_historical_close.side_effect = (
        lambda ticker, target_date, preferred_exchange=None: {
            date(2024, 1, 12): 100.0,  # 2024-01-13 (Sat) -> 2024-01-12 (Fri)
            date(2024, 7, 12): 100.0,
            date(2024, 12, 13): 100.0,
        }[target_date]
    )

    service = PriceService(price_client)

    change_rates = service.get_stock_change_rates("AAPL", as_of=date(2025, 1, 13))

    assert change_rates["1y"] == Decimal("20")
    price_client.get_historical_close.assert_any_call(
        "AAPL", date(2024, 1, 12), preferred_exchange=None
    )


def test_get_stock_change_rates_calls_price_client_historical_close():
    """PriceService는 통합 가격 클라이언트의 과거 종가 조회를 호출한다."""
    price_client = Mock()
    price_client.get_price.return_value = PriceQuote(
        symbol="AAPL", name="Apple Inc.", price=120.0, market="US", currency="USD"
    )
    price_client.get_historical_close.return_value = 100.0

    service = PriceService(price_client)

    change_rates = service.get_stock_change_rates("AAPL", as_of=date(2025, 1, 15))

    assert change_rates["1y"] == Decimal("20")
    price_client.get_historical_close.assert_any_call(
        "AAPL", date(2024, 1, 15), preferred_exchange=None
    )


def test_get_stock_price_returns_zero_when_overseas_price_errors():
    """해외 주식 조회가 연속 실패하면 0 가격으로 반환한다."""
    domestic_client = Mock()
    overseas_client = Mock()
    request = httpx.Request("GET", "https://example.com")
    response = httpx.Response(status_code=500, request=request)
    overseas_client.fetch_current_price.side_effect = [
        httpx.HTTPStatusError("Server error", request=request, response=response),
        httpx.HTTPStatusError("Server error", request=request, response=response),
        httpx.HTTPStatusError("Server error", request=request, response=response),
    ]
    unified_client = KisUnifiedPriceClient(domestic_client, overseas_client)

    service = PriceService(unified_client)

    price, currency, name, exchange = service.get_stock_price("SPY")

    assert price == Decimal("0")
    assert currency == "USD"
    assert name == ""
    assert exchange is None
    overseas_client.fetch_current_price.assert_any_call("NAS", "SPY")
    overseas_client.fetch_current_price.assert_any_call("NYS", "SPY")
    overseas_client.fetch_current_price.assert_any_call("AMS", "SPY")


def test_get_stock_price_returns_zero_when_domestic_price_errors():
    """국내 주식 조회가 실패하면 0 가격으로 반환한다."""
    domestic_client = Mock()
    overseas_client = Mock()
    request = httpx.Request("GET", "https://example.com")
    response = httpx.Response(status_code=500, request=request)
    domestic_client.fetch_current_price.side_effect = httpx.HTTPStatusError(
        "Server error", request=request, response=response
    )
    unified_client = KisUnifiedPriceClient(domestic_client, overseas_client)

    service = PriceService(unified_client)

    price, currency, name, exchange = service.get_stock_price("360750")

    assert price == Decimal("0")
    assert currency == "KRW"
    assert name == ""
    assert exchange is None
    domestic_client.fetch_current_price.assert_called_once_with("J", "360750")


def test_get_stock_change_rates_returns_zero_when_domestic_history_errors():
    """국내 과거 종가 조회 실패 시 변동률은 0으로 처리한다."""
    domestic_client = Mock()
    overseas_client = Mock()
    domestic_client.fetch_current_price.return_value = PriceQuote(
        symbol="360750",
        name="Test",
        price=1000.0,
        market="KR",
        currency="KRW",
    )
    request = httpx.Request("GET", "https://example.com")
    response = httpx.Response(status_code=500, request=request)
    domestic_client.fetch_historical_close.side_effect = httpx.HTTPStatusError(
        "Server error", request=request, response=response
    )
    unified_client = KisUnifiedPriceClient(domestic_client, overseas_client)

    service = PriceService(unified_client)

    change_rates = service.get_stock_change_rates("360750", as_of=date(2025, 1, 3))

    assert change_rates == {
        "1y": Decimal("0"),
        "6m": Decimal("0"),
        "1m": Decimal("0"),
    }


def test_get_stock_price_uses_cached_price_for_today():
    """오늘 캐시된 가격이 있으면 API 호출 없이 반환한다."""
    price_client = Mock()
    cache_repo = Mock()
    cache_repo.get_by_ticker_and_date.return_value = StockPrice(
        id=uuid4(),
        ticker="QQQ",
        price=Decimal("410.12"),
        currency="USD",
        name="Invesco QQQ Trust",
        exchange="NAS",
        price_date=date(2026, 1, 4),
        created_at=None,  # type: ignore[arg-type]
        updated_at=None,  # type: ignore[arg-type]
    )

    service = PriceService(
        price_client,
        price_cache_repository=cache_repo,
        today_provider=lambda: date(2026, 1, 4),
    )

    price, currency, name, exchange = service.get_stock_price("QQQ")

    assert price == Decimal("410.12")
    assert currency == "USD"
    assert name == "Invesco QQQ Trust"
    assert exchange == "NAS"
    price_client.get_price.assert_not_called()


def test_get_stock_price_saves_price_when_cache_miss():
    """캐시가 없으면 조회한 가격을 저장한다."""
    price_client = Mock()
    price_client.get_price.return_value = PriceQuote(
        symbol="QQQ",
        name="Invesco QQQ Trust",
        price=410.12,
        market="US",
        currency="USD",
        exchange="NAS",
    )
    cache_repo = Mock()

    def cache_lookup(ticker, price_date):
        if price_date == date(2026, 1, 8):
            return StockPrice(
                id=uuid4(),
                ticker="AAPL",
                price=Decimal("120.0"),
                currency="USD",
                name="Apple Inc.",
                exchange="NAS",
                price_date=price_date,
                created_at=None,  # type: ignore[arg-type]
                updated_at=None,  # type: ignore[arg-type]
            )
        return None

    cache_repo.get_by_ticker_and_date.side_effect = cache_lookup

    service = PriceService(
        price_client,
        price_cache_repository=cache_repo,
        today_provider=lambda: date(2026, 1, 4),
    )

    price, currency, name, exchange = service.get_stock_price("QQQ")

    assert price == Decimal("410.12")
    assert currency == "USD"
    assert name == "Invesco QQQ Trust"
    assert exchange == "NAS"
    cache_repo.save.assert_called_once_with(
        ticker="QQQ",
        price_date=date(2026, 1, 4),
        price=Decimal("410.12"),
        currency="USD",
        name="Invesco QQQ Trust",
        exchange="NAS",
    )


def test_get_stock_change_rates_uses_cached_historical_prices():
    """과거 종가 캐시가 있으면 API 호출 없이 변동률을 계산한다."""
    price_client = Mock()
    price_client.get_price.return_value = PriceQuote(
        symbol="AAPL",
        name="Apple Inc.",
        price=120.0,
        market="US",
        currency="USD",
        exchange="NAS",
    )
    cache_repo = Mock()

    as_of = date(2026, 1, 8)
    one_year = date(2025, 1, 8)
    six_months = date(2025, 7, 8)
    one_month = date(2025, 12, 8)

    def cache_lookup(ticker, price_date):
        if price_date == one_year:
            return StockPrice(
                id=uuid4(),
                ticker="AAPL",
                price=Decimal("100.0"),
                currency="USD",
                name="Apple Inc.",
                exchange="NAS",
                price_date=price_date,
                created_at=None,  # type: ignore[arg-type]
                updated_at=None,  # type: ignore[arg-type]
            )
        if price_date == six_months:
            return StockPrice(
                id=uuid4(),
                ticker="AAPL",
                price=Decimal("80.0"),
                currency="USD",
                name="Apple Inc.",
                exchange="NAS",
                price_date=price_date,
                created_at=None,  # type: ignore[arg-type]
                updated_at=None,  # type: ignore[arg-type]
            )
        if price_date == one_month:
            return StockPrice(
                id=uuid4(),
                ticker="AAPL",
                price=Decimal("60.0"),
                currency="USD",
                name="Apple Inc.",
                exchange="NAS",
                price_date=price_date,
                created_at=None,  # type: ignore[arg-type]
                updated_at=None,  # type: ignore[arg-type]
            )
        return None

    cache_repo.get_by_ticker_and_date.side_effect = cache_lookup

    service = PriceService(
        price_client,
        price_cache_repository=cache_repo,
        today_provider=lambda: date(2026, 1, 8),
    )

    change_rates = service.get_stock_change_rates("AAPL", as_of=as_of)

    assert change_rates["1y"] == Decimal("20")
    assert change_rates["6m"] == Decimal("50")
    assert change_rates["1m"] == Decimal("100")
    price_client.get_historical_close.assert_not_called()


def test_get_stock_change_rates_skips_cache_save_on_history_error():
    """과거 종가 조회 오류 시 캐시 저장 없이 0으로 처리한다."""
    price_client = Mock()
    price_client.get_price.return_value = PriceQuote(
        symbol="AAPL",
        name="Apple Inc.",
        price=120.0,
        market="US",
        currency="USD",
        exchange="NAS",
    )
    price_client.get_historical_close.side_effect = RuntimeError("boom")
    cache_repo = Mock()

    def cache_lookup(ticker, price_date):
        if price_date == date(2026, 1, 8):
            return StockPrice(
                id=uuid4(),
                ticker="AAPL",
                price=Decimal("120.0"),
                currency="USD",
                name="Apple Inc.",
                exchange="NAS",
                price_date=price_date,
                created_at=None,  # type: ignore[arg-type]
                updated_at=None,  # type: ignore[arg-type]
            )
        return None

    cache_repo.get_by_ticker_and_date.side_effect = cache_lookup

    service = PriceService(
        price_client,
        price_cache_repository=cache_repo,
        today_provider=lambda: date(2026, 1, 8),
    )

    change_rates = service.get_stock_change_rates("AAPL", as_of=date(2026, 1, 8))

    assert change_rates == {
        "1y": Decimal("0"),
        "6m": Decimal("0"),
        "1m": Decimal("0"),
    }
    cache_repo.save.assert_not_called()
