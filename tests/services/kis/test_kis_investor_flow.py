import httpx

from portfolio_manager.services.kis.kis_domestic_investor_client import (
    DomesticInvestorFlow,
    KisDomesticInvestorClient,
)


def _make_client(handler) -> KisDomesticInvestorClient:
    transport = httpx.MockTransport(handler)
    http = httpx.Client(
        transport=transport, base_url="https://openapi.koreainvestment.com:9443"
    )
    return KisDomesticInvestorClient(
        client=http,
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        tr_id="FHKST01010900",
        cust_type="P",
    )


def test_investor_flow_request_uses_headers_and_params():
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
                "output": [
                    {
                        "stck_bsop_date": "20260503",
                        "frgn_ntby_qty": "1000",
                        "orgn_ntby_qty": "-500",
                        "frgn_ntby_tr_pbmn": "5000000",
                        "orgn_ntby_tr_pbmn": "-2500000",
                    },
                    {
                        "stck_bsop_date": "20260502",
                        "frgn_ntby_qty": "200",
                        "orgn_ntby_qty": "300",
                        "frgn_ntby_tr_pbmn": "1000000",
                        "orgn_ntby_tr_pbmn": "1500000",
                    },
                ],
            },
        )

    kis = _make_client(handler)
    result = kis.fetch_daily_flow("005930")

    assert captured["method"] == "GET"
    assert captured["path"] == "/uapi/domestic-stock/v1/quotations/inquire-investor"
    assert captured["params"] == {
        "FID_COND_MRKT_DIV_CODE": "J",
        "FID_INPUT_ISCD": "005930",
    }
    assert captured["headers"]["authorization"] == "Bearer access-token"
    assert captured["headers"]["appkey"] == "app-key"
    assert captured["headers"]["appsecret"] == "app-secret"
    assert captured["headers"]["tr_id"] == "FHKST01010900"
    assert captured["headers"]["custtype"] == "P"

    assert result == [
        DomesticInvestorFlow(
            date="20260503",
            foreign_net_qty=1000,
            institution_net_qty=-500,
            foreign_net_krw=5000000,
            institution_net_krw=-2500000,
        ),
        DomesticInvestorFlow(
            date="20260502",
            foreign_net_qty=200,
            institution_net_qty=300,
            foreign_net_krw=1000000,
            institution_net_krw=1500000,
        ),
    ]


def test_investor_flow_handles_empty_output():
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            status_code=200,
            json={"rt_cd": "0", "output": []},
        )

    kis = _make_client(handler)
    result = kis.fetch_daily_flow("005930")

    assert result == []


def test_investor_flow_handles_missing_optional_fields():
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            status_code=200,
            json={
                "rt_cd": "0",
                "output": [
                    {
                        "stck_bsop_date": "20260501",
                        "frgn_ntby_qty": "50",
                        # orgn_ntby_qty, frgn_ntby_tr_pbmn, orgn_ntby_tr_pbmn absent
                    }
                ],
            },
        )

    kis = _make_client(handler)
    result = kis.fetch_daily_flow("000660")

    assert result == [
        DomesticInvestorFlow(
            date="20260501",
            foreign_net_qty=50,
            institution_net_qty=0,
            foreign_net_krw=0,
            institution_net_krw=0,
        )
    ]
