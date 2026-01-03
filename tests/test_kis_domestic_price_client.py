import httpx

from portfolio_manager.services.kis_domestic_price_client import (
    KisDomesticPriceClient,
    PriceQuote,
)


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
    client = httpx.Client(transport=transport, base_url="https://openapi.koreainvestment.com:9443")

    kis = KisDomesticPriceClient(
        client=client,
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        cust_type="P",
        env="real",
    )

    result = kis.fetch_current_price(fid_cond_mrkt_div_code="J", fid_input_iscd="005930")

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
    )
