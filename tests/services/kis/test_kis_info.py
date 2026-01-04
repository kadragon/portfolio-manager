import httpx

from portfolio_manager.services.kis.kis_domestic_info_client import (
    DomesticStockInfo,
    KisDomesticInfoClient,
)


def test_domestic_info_request_uses_headers_and_params():
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
                    "pdno": "005930",
                    "prdt_type_cd": "300",
                    "mket_id_cd": "STK",
                    "prdt_name": "삼성전자",
                },
            },
        )

    transport = httpx.MockTransport(handler)
    client = httpx.Client(
        transport=transport, base_url="https://openapi.koreainvestment.com:9443"
    )

    kis = KisDomesticInfoClient(
        client=client,
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        tr_id="CTPF1002R",
        cust_type="P",
    )

    result = kis.fetch_basic_info(prdt_type_cd="300", pdno="005930")

    assert captured["method"] == "GET"
    assert captured["path"] == "/uapi/domestic-stock/v1/quotations/search-stock-info"
    assert captured["params"] == {"PRDT_TYPE_CD": "300", "PDNO": "005930"}
    assert captured["headers"]["authorization"] == "Bearer access-token"
    assert captured["headers"]["appkey"] == "app-key"
    assert captured["headers"]["appsecret"] == "app-secret"
    assert captured["headers"]["tr_id"] == "CTPF1002R"
    assert captured["headers"]["custtype"] == "P"

    assert result == DomesticStockInfo(
        pdno="005930",
        prdt_type_cd="300",
        market_id="STK",
        name="삼성전자",
    )


def test_domestic_info_uses_fallback_name_fields():
    captured = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["method"] = request.method
        return httpx.Response(
            status_code=200,
            json={
                "rt_cd": "0",
                "msg_cd": "",
                "msg1": "",
                "output": {
                    "pdno": "310970",
                    "prdt_type_cd": "300",
                    "mket_id_cd": "STK",
                    "prdt_name": "",
                    "prdt_name120": "KB RISE 미국",
                    "prdt_abrv_name": "KB RISE",
                },
            },
        )

    transport = httpx.MockTransport(handler)
    client = httpx.Client(
        transport=transport, base_url="https://openapi.koreainvestment.com:9443"
    )

    kis = KisDomesticInfoClient(
        client=client,
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        tr_id="CTPF1002R",
        cust_type="P",
    )

    result = kis.fetch_basic_info(prdt_type_cd="300", pdno="310970")

    assert captured["method"] == "GET"
    assert result.name == "KB RISE 미국"
