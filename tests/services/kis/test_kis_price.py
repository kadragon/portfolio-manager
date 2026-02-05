from datetime import date
from unittest.mock import MagicMock, call

import httpx
import pytest

from portfolio_manager.services.kis.kis_domestic_price_client import (
    KisDomesticPriceClient,
    PriceQuote,
)
from portfolio_manager.services.kis.kis_overseas_price_client import (
    KisOverseasPriceClient,
)
from portfolio_manager.services.kis.kis_unified_price_client import (
    KisUnifiedPriceClient,
)
from portfolio_manager.services.kis.kis_domestic_info_client import DomesticStockInfo
from portfolio_manager.services.kis.kis_price_parser import (
    parse_korea_price,
    parse_us_price,
)


# --- KisDomesticPriceClient Tests ---


def test_domestic_price_request_uses_headers_and_params():
    captured = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["method"] = request.method
        captured["path"] = request.url.path
        captured["params"] = dict(request.url.params)
        captured["headers"] = dict(request.headers)
        return httpx.Response(
            status_code=200,
            json={
                "rt_cd": "0",
                "msg_cd": "",
                "msg1": "",
                "output": {
                    "stck_prpr": "73500",
                    "hts_kor_isnm": "삼성전자",
                },
            },
        )

    transport = httpx.MockTransport(handler)
    client = httpx.Client(
        transport=transport, base_url="https://openapi.koreainvestment.com:9443"
    )

    kis = KisDomesticPriceClient(
        client=client,
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        cust_type="P",
        env="real",
    )

    result = kis.fetch_current_price(
        fid_cond_mrkt_div_code="J", fid_input_iscd="005930"
    )

    assert captured["method"] == "GET"
    assert captured["path"] == "/uapi/domestic-stock/v1/quotations/inquire-price"
    assert captured["params"] == {
        "FID_COND_MRKT_DIV_CODE": "J",
        "FID_INPUT_ISCD": "005930",
    }
    assert captured["headers"]["authorization"] == "Bearer access-token"
    assert captured["headers"]["appkey"] == "app-key"
    assert captured["headers"]["appsecret"] == "app-secret"
    assert captured["headers"]["tr_id"] == "FHKST01010100"
    assert captured["headers"]["custtype"] == "P"

    assert result == PriceQuote(
        symbol="005930",
        name="삼성전자",
        price=73500,
        market="KR",
        currency="KRW",
    )


def test_env_normalization_allows_whitespace_and_case():
    assert KisDomesticPriceClient._tr_id_for_env(" REAL ") == "FHKST01010100"
    assert KisDomesticPriceClient._tr_id_for_env("VPS ") == "FHKST01010100"


def test_env_allows_real_prod_token():
    assert KisDomesticPriceClient._tr_id_for_env("real/prod") == "FHKST01010100"


# --- KisOverseasPriceClient Tests ---


def test_overseas_price_empty_last_returns_zero():
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            status_code=200,
            json={
                "output": {
                    "last": "",
                    "symbol": "AAPL",
                    "name": "Apple Inc",
                }
            },
        )

    transport = httpx.MockTransport(handler)
    client = httpx.Client(
        transport=transport, base_url="https://openapi.koreainvestment.com:9443"
    )

    kis = KisOverseasPriceClient(
        client=client,
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        cust_type="P",
        env="real",
    )

    result = kis.fetch_current_price(excd="NAS", symb="AAPL")

    assert result.price == 0.0
    assert result.symbol == "AAPL"
    assert result.name == "Apple Inc"


def test_overseas_price_uses_alternate_name_field():
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            status_code=200,
            json={
                "output": {
                    "last": "144.76",
                    "symbol": "VYM",
                    "enname": "Vanguard High Dividend Yield ETF",
                }
            },
        )

    transport = httpx.MockTransport(handler)
    client = httpx.Client(
        transport=transport, base_url="https://openapi.koreainvestment.com:9443"
    )

    kis = KisOverseasPriceClient(
        client=client,
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        cust_type="P",
        env="real",
    )

    result = kis.fetch_current_price(excd="NAS", symb="VYM")

    assert result.name == "Vanguard High Dividend Yield ETF"


# --- KisUnifiedPriceClient Tests ---


@pytest.fixture
def mock_domestic_client():
    client = MagicMock()
    client.fetch_current_price.return_value = PriceQuote(
        symbol="005930", name="삼성전자", price=70000, market="KR", currency="KRW"
    )
    return client


@pytest.fixture
def mock_overseas_client():
    client = MagicMock()
    client.fetch_current_price.return_value = PriceQuote(
        symbol="AAPL", name="Apple Inc.", price=150.0, market="US", currency="USD"
    )
    return client


@pytest.fixture
def mock_domestic_info_client():
    client = MagicMock()
    client.fetch_basic_info.return_value = DomesticStockInfo(
        pdno="005930", prdt_type_cd="300", market_id="STK", name="삼성전자"
    )
    return client


def test_detects_6_digit_numeric_ticker_as_domestic(
    mock_domestic_client, mock_overseas_client
):
    """6자리 숫자 티커는 국내 주식으로 처리한다."""
    unified = KisUnifiedPriceClient(mock_domestic_client, mock_overseas_client)

    unified.get_price("005930")

    mock_domestic_client.fetch_current_price.assert_called_once_with("J", "005930")
    mock_overseas_client.fetch_current_price.assert_not_called()


def test_detects_alphabetic_ticker_as_overseas(
    mock_domestic_client, mock_overseas_client
):
    """알파벳 티커는 해외 주식으로 처리한다."""
    unified = KisUnifiedPriceClient(mock_domestic_client, mock_overseas_client)

    unified.get_price("AAPL")

    mock_overseas_client.fetch_current_price.assert_called_once_with("NAS", "AAPL")
    mock_domestic_client.fetch_current_price.assert_not_called()


def test_overseas_price_falls_back_to_additional_exchange_when_empty(
    mock_domestic_client,
):
    """해외 시세가 비어있으면 다른 거래소로 재시도한다."""
    overseas_client = MagicMock()
    overseas_client.fetch_current_price.side_effect = [
        PriceQuote(symbol="SPY", name="", price=0.0, market="US", currency="USD"),
        PriceQuote(symbol="SPY", name="", price=0.0, market="US", currency="USD"),
        PriceQuote(
            symbol="SPY",
            name="SPDR S&P 500 ETF Trust",
            price=470.0,
            market="US",
            currency="USD",
        ),
    ]
    unified = KisUnifiedPriceClient(mock_domestic_client, overseas_client)

    quote = unified.get_price("SPY")

    assert quote.name == "SPDR S&P 500 ETF Trust"
    assert quote.price == 470.0
    assert overseas_client.fetch_current_price.call_count == 3
    overseas_client.fetch_current_price.assert_has_calls(
        [call("NAS", "SPY"), call("NYS", "SPY"), call("AMS", "SPY")]
    )


def test_overseas_price_falls_back_when_name_missing(
    mock_domestic_client,
):
    """해외 종목명 누락 시 다른 거래소 응답으로 보완한다."""
    overseas_client = MagicMock()
    overseas_client.fetch_current_price.side_effect = [
        PriceQuote(symbol="VYM", name="", price=144.76, market="US", currency="USD"),
        PriceQuote(
            symbol="VYM",
            name="Vanguard High Dividend Yield ETF",
            price=144.76,
            market="US",
            currency="USD",
        ),
    ]
    unified = KisUnifiedPriceClient(mock_domestic_client, overseas_client)

    quote = unified.get_price("VYM")

    assert quote.name == "Vanguard High Dividend Yield ETF"
    overseas_client.fetch_current_price.assert_has_calls(
        [call("NAS", "VYM"), call("NYS", "VYM")]
    )


def test_overseas_price_skips_http_errors_and_tries_next_exchange(
    mock_domestic_client,
):
    """해외 시세 조회가 HTTP 에러면 다음 거래소로 넘어간다."""
    overseas_client = MagicMock()
    request = httpx.Request("GET", "https://example.com")
    response = httpx.Response(status_code=500, request=request)
    overseas_client.fetch_current_price.side_effect = [
        httpx.HTTPStatusError("Server error", request=request, response=response),
        PriceQuote(
            symbol="SCHD",
            name="Schwab US Dividend Equity ETF",
            price=70.0,
            market="US",
            currency="USD",
        ),
    ]
    unified = KisUnifiedPriceClient(mock_domestic_client, overseas_client)

    quote = unified.get_price("SCHD")

    assert quote.name == "Schwab US Dividend Equity ETF"
    overseas_client.fetch_current_price.assert_has_calls(
        [call("NAS", "SCHD"), call("NYS", "SCHD")]
    )


def test_overseas_price_returns_zero_when_all_exchanges_fail(
    mock_domestic_client,
):
    """모든 거래소가 실패하면 0 가격의 기본 응답을 반환한다."""
    overseas_client = MagicMock()
    request = httpx.Request("GET", "https://example.com")
    response = httpx.Response(status_code=500, request=request)
    overseas_client.fetch_current_price.side_effect = [
        httpx.HTTPStatusError("Server error", request=request, response=response),
        httpx.HTTPStatusError("Server error", request=request, response=response),
        httpx.HTTPStatusError("Server error", request=request, response=response),
    ]
    unified = KisUnifiedPriceClient(mock_domestic_client, overseas_client)

    quote = unified.get_price("SPY")

    assert quote.symbol == "SPY"
    assert quote.name == ""
    assert quote.price == 0.0
    assert quote.currency == "USD"
    assert quote.exchange is None
    assert overseas_client.fetch_current_price.call_count == 3


def test_domestic_price_returns_zero_when_http_error(
    mock_overseas_client,
):
    """국내 시세 조회가 HTTP 에러면 기본 응답을 반환한다."""
    domestic_client = MagicMock()
    request = httpx.Request("GET", "https://example.com")
    response = httpx.Response(status_code=500, request=request)
    domestic_client.fetch_current_price.side_effect = httpx.HTTPStatusError(
        "Server error", request=request, response=response
    )
    unified = KisUnifiedPriceClient(domestic_client, mock_overseas_client)

    quote = unified.get_price("360750")

    assert quote.symbol == "360750"
    assert quote.name == ""
    assert quote.price == 0.0
    assert quote.currency == "KRW"
    assert quote.exchange is None
    domestic_client.fetch_current_price.assert_called_once_with("J", "360750")


def test_detects_6_digit_alphanumeric_ticker_as_domestic(
    mock_domestic_client, mock_overseas_client
):
    """6자리 영숫자 혼합 티커는 국내 주식으로 처리한다 (예: 0052D0)."""
    unified = KisUnifiedPriceClient(mock_domestic_client, mock_overseas_client)

    unified.get_price("0052D0")

    mock_domestic_client.fetch_current_price.assert_called_once_with("J", "0052D0")
    mock_overseas_client.fetch_current_price.assert_not_called()


def test_domestic_name_falls_back_to_basic_info_when_missing(
    mock_domestic_client, mock_overseas_client, mock_domestic_info_client
):
    """국내 가격 응답에 이름이 없으면 기본정보로 보완한다."""
    mock_domestic_client.fetch_current_price.return_value = PriceQuote(
        symbol="005930", name="", price=70000, market="KR", currency="KRW"
    )
    unified = KisUnifiedPriceClient(
        mock_domestic_client, mock_overseas_client, mock_domestic_info_client
    )

    quote = unified.get_price("005930")

    mock_domestic_client.fetch_current_price.assert_called_once_with("J", "005930")
    mock_domestic_info_client.fetch_basic_info.assert_called_once_with(
        prdt_type_cd="300", pdno="005930"
    )
    assert quote.name == "삼성전자"


def test_domestic_name_uses_configured_product_type_code(
    mock_domestic_client, mock_overseas_client, mock_domestic_info_client
):
    """기본정보 조회 시 설정된 상품유형코드를 사용한다."""
    mock_domestic_client.fetch_current_price.return_value = PriceQuote(
        symbol="005930", name="", price=70000, market="KR", currency="KRW"
    )
    unified = KisUnifiedPriceClient(
        mock_domestic_client,
        mock_overseas_client,
        mock_domestic_info_client,
        prdt_type_cd="301",
    )

    unified.get_price("005930")

    mock_domestic_info_client.fetch_basic_info.assert_called_once_with(
        prdt_type_cd="301", pdno="005930"
    )


def test_domestic_name_lookup_failure_returns_price_quote(
    mock_domestic_client, mock_overseas_client, mock_domestic_info_client
):
    """기본정보 조회가 실패해도 가격 응답은 반환한다."""
    mock_domestic_client.fetch_current_price.return_value = PriceQuote(
        symbol="005930", name="", price=70000, market="KR", currency="KRW"
    )
    mock_domestic_info_client.fetch_basic_info.side_effect = KeyError("output")
    unified = KisUnifiedPriceClient(
        mock_domestic_client, mock_overseas_client, mock_domestic_info_client
    )

    quote = unified.get_price("005930")

    assert quote.name == ""


def test_unified_historical_close_routes_domestic_by_length(
    mock_domestic_client, mock_overseas_client
):
    """과거 종가는 6자리 티커면 국내 클라이언트로 라우팅한다."""
    mock_domestic_client.fetch_historical_close.return_value = 70000
    unified = KisUnifiedPriceClient(mock_domestic_client, mock_overseas_client)

    close_price = unified.get_historical_close("005930", target_date=date(2024, 1, 15))

    assert close_price == 70000
    mock_domestic_client.fetch_historical_close.assert_called_once_with(
        fid_input_iscd="005930", target_date=date(2024, 1, 15)
    )
    mock_overseas_client.fetch_historical_close.assert_not_called()


def test_unified_historical_close_checks_overseas_exchanges(
    mock_domestic_client, mock_overseas_client
):
    """해외 종가는 거래소를 순차 조회하며 첫 유효 값을 반환한다."""
    mock_overseas_client.fetch_historical_close.side_effect = [
        0.0,
        0.0,
        150.0,
    ]
    unified = KisUnifiedPriceClient(mock_domestic_client, mock_overseas_client)

    close_price = unified.get_historical_close("AAPL", target_date=date(2024, 1, 15))

    assert close_price == 150.0
    assert mock_overseas_client.fetch_historical_close.call_count == 3
    mock_overseas_client.fetch_historical_close.assert_has_calls(
        [
            call(excd="NAS", symb="AAPL", target_date=date(2024, 1, 15)),
            call(excd="NYS", symb="AAPL", target_date=date(2024, 1, 15)),
            call(excd="AMS", symb="AAPL", target_date=date(2024, 1, 15)),
        ]
    )


def test_unified_price_uses_preferred_exchange_first(
    mock_domestic_client, mock_overseas_client
):
    """저장된 거래소가 있으면 해당 거래소를 먼저 조회한다."""
    mock_overseas_client.fetch_current_price.return_value = PriceQuote(
        symbol="SCHD",
        name="",
        price=70.0,
        market="US",
        currency="USD",
    )
    unified = KisUnifiedPriceClient(mock_domestic_client, mock_overseas_client)

    quote = unified.get_price("SCHD", preferred_exchange="NYS")

    assert quote.price == 70.0
    mock_overseas_client.fetch_current_price.assert_called_once_with("NYS", "SCHD")


def test_unified_price_does_not_fallback_when_preferred_exchange_succeeds(
    mock_domestic_client, mock_overseas_client
):
    """선호 거래소가 가격을 반환하면 추가 거래소는 조회하지 않는다."""
    mock_overseas_client.fetch_current_price.return_value = PriceQuote(
        symbol="SCHD",
        name="",
        price=70.0,
        market="US",
        currency="USD",
    )
    unified = KisUnifiedPriceClient(mock_domestic_client, mock_overseas_client)

    quote = unified.get_price("SCHD", preferred_exchange="NYS")

    assert quote.price == 70.0
    assert mock_overseas_client.fetch_current_price.call_count == 1
    mock_overseas_client.fetch_current_price.assert_called_once_with("NYS", "SCHD")


def test_unified_historical_close_uses_preferred_exchange_first(
    mock_domestic_client, mock_overseas_client
):
    """과거 종가도 저장된 거래소를 먼저 조회한다."""
    mock_overseas_client.fetch_historical_close.side_effect = [150.0]
    unified = KisUnifiedPriceClient(mock_domestic_client, mock_overseas_client)

    close_price = unified.get_historical_close(
        "AAPL", target_date=date(2024, 1, 15), preferred_exchange="NYS"
    )

    assert close_price == 150.0
    mock_overseas_client.fetch_historical_close.assert_called_once_with(
        excd="NYS", symb="AAPL", target_date=date(2024, 1, 15)
    )


def test_unified_historical_close_does_not_fallback_when_preferred_exchange_succeeds(
    mock_domestic_client, mock_overseas_client
):
    """선호 거래소가 과거 종가를 반환하면 추가 거래소는 조회하지 않는다."""
    mock_overseas_client.fetch_historical_close.return_value = 150.0
    unified = KisUnifiedPriceClient(mock_domestic_client, mock_overseas_client)

    close_price = unified.get_historical_close(
        "AAPL", target_date=date(2024, 1, 15), preferred_exchange="NYS"
    )

    assert close_price == 150.0
    assert mock_overseas_client.fetch_historical_close.call_count == 1
    mock_overseas_client.fetch_historical_close.assert_called_once_with(
        excd="NYS", symb="AAPL", target_date=date(2024, 1, 15)
    )


def test_unified_historical_close_does_not_fallback_when_preferred_returns_zero(
    mock_domestic_client, mock_overseas_client
):
    """선호 거래소가 0을 반환해도 추가 거래소는 조회하지 않는다."""
    mock_overseas_client.fetch_historical_close.return_value = 0.0
    unified = KisUnifiedPriceClient(mock_domestic_client, mock_overseas_client)

    close_price = unified.get_historical_close(
        "AAPL", target_date=date(2024, 1, 15), preferred_exchange="NYS"
    )

    assert close_price == 0.0
    assert mock_overseas_client.fetch_historical_close.call_count == 1
    mock_overseas_client.fetch_historical_close.assert_called_once_with(
        excd="NYS", symb="AAPL", target_date=date(2024, 1, 15)
    )


def test_unified_historical_close_skips_http_errors(
    mock_domestic_client, mock_overseas_client
):
    """과거 종가 조회가 HTTP 에러면 다음 거래소로 넘어간다."""
    request = httpx.Request("GET", "https://example.com")
    response = httpx.Response(status_code=500, request=request)
    mock_overseas_client.fetch_historical_close.side_effect = [
        httpx.HTTPStatusError("Server error", request=request, response=response),
        0.0,
        150.0,
    ]
    unified = KisUnifiedPriceClient(mock_domestic_client, mock_overseas_client)

    close_price = unified.get_historical_close("AAPL", target_date=date(2024, 1, 15))

    assert close_price == 150.0
    assert mock_overseas_client.fetch_historical_close.call_count == 3


def test_unified_historical_close_returns_zero_when_domestic_http_error(
    mock_overseas_client,
):
    """국내 과거 종가 조회가 HTTP 에러면 0을 반환한다."""
    domestic_client = MagicMock()
    request = httpx.Request("GET", "https://example.com")
    response = httpx.Response(status_code=500, request=request)
    domestic_client.fetch_historical_close.side_effect = httpx.HTTPStatusError(
        "Server error", request=request, response=response
    )
    unified = KisUnifiedPriceClient(domestic_client, mock_overseas_client)

    close_price = unified.get_historical_close("360750", target_date=date(2025, 1, 3))

    assert close_price == 0.0
    domestic_client.fetch_historical_close.assert_called_once_with(
        fid_input_iscd="360750", target_date=date(2025, 1, 3)
    )


# --- Price Parser Tests ---


def test_parse_korea_price_returns_common_model():
    payload = {
        "output": {
            "stck_prpr": "73500",
            "stck_code": "005930",
            "hts_kor_isnm": "삼성전자",
        }
    }

    price = parse_korea_price(payload)

    assert price.symbol == "005930"
    assert price.name == "삼성전자"
    assert price.price == 73500
    assert price.market == "KR"


def test_parse_us_price_returns_common_model():
    payload = {
        "output": {
            "last": "192.45",
            "symbol": "AAPL",
            "name": "Apple Inc",
        }
    }

    price = parse_us_price(payload)

    assert price.symbol == "AAPL"
    assert price.name == "Apple Inc"
    assert price.price == 192.45
    assert price.market == "US"
