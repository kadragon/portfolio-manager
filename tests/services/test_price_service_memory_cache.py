"""Test PriceService in-memory caching."""

from decimal import Decimal
from datetime import date
from unittest.mock import Mock

from portfolio_manager.services.kis.kis_price_parser import PriceQuote
from portfolio_manager.services.price_service import PriceService


def test_get_stock_price_uses_memory_cache():
    """두 번째 호출 시 메모리 캐시를 사용해 API를 호출하지 않는다."""
    # Given: Mock price client
    price_client = Mock()
    price_client.get_price.return_value = PriceQuote(
        symbol="AAPL",
        name="Apple Inc.",
        price=150.0,
        market="US",
        currency="USD",
        exchange="NAS",
    )

    service = PriceService(price_client)

    # When: 같은 티커로 두 번 조회
    result1 = service.get_stock_price("AAPL")
    result2 = service.get_stock_price("AAPL")

    # Then: API는 한 번만 호출되고 결과는 동일
    price_client.get_price.assert_called_once()
    assert result1 == result2
    assert result1 == (Decimal("150.0"), "USD", "Apple Inc.", "NAS")


def test_get_stock_change_rates_uses_memory_cache():
    """변동률 두 번째 호출 시 메모리 캐시를 사용해 API를 호출하지 않는다."""
    # Given: Mock price client
    price_client = Mock()
    price_client.get_price.return_value = PriceQuote(
        symbol="AAPL",
        name="Apple Inc.",
        price=120.0,
        market="US",
        currency="USD",
        exchange="NAS",
    )
    price_client.get_historical_close.side_effect = (
        lambda ticker, target_date, preferred_exchange=None: {
            date(2024, 1, 15): 100.0,
            date(2024, 7, 15): 80.0,
            date(2024, 12, 13): 60.0,
        }[target_date]
    )

    service = PriceService(price_client)
    as_of = date(2025, 1, 15)

    # When: 같은 티커로 두 번 변동률 조회
    result1 = service.get_stock_change_rates("AAPL", as_of=as_of)
    result2 = service.get_stock_change_rates("AAPL", as_of=as_of)

    # Then: historical_close API는 첫 번째 호출에서만 실행
    assert price_client.get_historical_close.call_count == 3  # 1Y, 6M, 1M
    assert result1 == result2
    assert result1["1y"] == Decimal("20")
    assert result1["6m"] == Decimal("50")
    assert result1["1m"] == Decimal("100")


def test_get_stock_price_cache_keys_by_exchange():
    """다른 preferred_exchange로 조회하면 각각 별도로 API를 호출한다."""
    # Given: Mock price client that returns different prices per exchange
    price_client = Mock()

    def get_price_side_effect(ticker, preferred_exchange=None):
        if preferred_exchange == "NAS":
            return PriceQuote(
                symbol="AAPL",
                name="Apple Inc.",
                price=150.0,
                market="US",
                currency="USD",
                exchange="NAS",
            )
        else:
            return PriceQuote(
                symbol="AAPL",
                name="Apple Inc.",
                price=151.0,
                market="US",
                currency="USD",
                exchange="NYS",
            )

    price_client.get_price.side_effect = get_price_side_effect

    service = PriceService(price_client)

    # When: 같은 티커를 다른 거래소로 조회
    result_nas = service.get_stock_price("AAPL", preferred_exchange="NAS")
    result_nys = service.get_stock_price("AAPL", preferred_exchange="NYS")

    # Then: 각각 API를 호출하고 다른 결과 반환
    assert price_client.get_price.call_count == 2
    assert result_nas[0] == Decimal("150.0")
    assert result_nas[3] == "NAS"
    assert result_nys[0] == Decimal("151.0")
    assert result_nys[3] == "NYS"


def test_get_stock_price_does_not_cache_zero_price():
    """가격이 0이면 메모리 캐시에 저장하지 않고 다음 호출에서 재시도한다."""
    # Given: Mock price client that returns 0 first, then valid price
    price_client = Mock()
    call_count = [0]

    def get_price_side_effect(ticker, preferred_exchange=None):
        call_count[0] += 1
        if call_count[0] == 1:
            return PriceQuote(
                symbol="AAPL",
                name="",
                price=0.0,
                market="US",
                currency="USD",
                exchange=None,
            )
        return PriceQuote(
            symbol="AAPL",
            name="Apple Inc.",
            price=150.0,
            market="US",
            currency="USD",
            exchange="NAS",
        )

    price_client.get_price.side_effect = get_price_side_effect

    service = PriceService(price_client)

    # When: 첫 번째 호출은 0 반환, 두 번째 호출
    result1 = service.get_stock_price("AAPL")
    result2 = service.get_stock_price("AAPL")

    # Then: 0은 캐시되지 않아 두 번째 호출도 API 호출
    assert price_client.get_price.call_count == 2
    assert result1[0] == Decimal("0")
    assert result2[0] == Decimal("150.0")


def test_get_stock_change_rates_cache_keys_by_exchange():
    """다른 preferred_exchange로 변동률 조회하면 각각 별도로 API를 호출한다."""
    # Given: Mock price client
    price_client = Mock()

    def get_price_side_effect(ticker, preferred_exchange=None):
        if preferred_exchange == "NAS":
            return PriceQuote(
                symbol="AAPL",
                name="Apple Inc.",
                price=120.0,
                market="US",
                currency="USD",
                exchange="NAS",
            )
        else:
            return PriceQuote(
                symbol="AAPL",
                name="Apple Inc.",
                price=130.0,
                market="US",
                currency="USD",
                exchange="NYS",
            )

    price_client.get_price.side_effect = get_price_side_effect
    price_client.get_historical_close.return_value = 100.0

    service = PriceService(price_client)
    as_of = date(2025, 1, 15)

    # When: 같은 티커를 다른 거래소로 변동률 조회
    result_nas = service.get_stock_change_rates(
        "AAPL", as_of=as_of, preferred_exchange="NAS"
    )
    result_nys = service.get_stock_change_rates(
        "AAPL", as_of=as_of, preferred_exchange="NYS"
    )

    # Then: 각각 API를 호출하고 다른 변동률 반환
    assert price_client.get_historical_close.call_count == 6  # 3 * 2 exchanges
    assert result_nas["1y"] == Decimal("20")  # (120-100)/100*100
    assert result_nys["1y"] == Decimal("30")  # (130-100)/100*100
