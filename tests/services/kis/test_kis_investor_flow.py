import pytest
import httpx

from portfolio_manager.services.kis.kis_api_error import KisApiBusinessError
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
        cust_type="P",
        env="real",
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
            individual_net_qty=0,
            foreign_net_krw=5000000,
            institution_net_krw=-2500000,
            individual_net_krw=0,
        ),
        DomesticInvestorFlow(
            date="20260502",
            foreign_net_qty=200,
            institution_net_qty=300,
            individual_net_qty=0,
            foreign_net_krw=1000000,
            institution_net_krw=1500000,
            individual_net_krw=0,
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
            individual_net_qty=0,
            foreign_net_krw=0,
            institution_net_krw=0,
            individual_net_krw=0,
        )
    ]


def test_investor_flow_raises_on_business_error():
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            status_code=200,
            json={"rt_cd": "1", "msg_cd": "EGW00123", "msg1": "token expired"},
        )

    kis = _make_client(handler)
    with pytest.raises(KisApiBusinessError) as exc_info:
        kis.fetch_daily_flow("005930")

    assert exc_info.value.code == "EGW00123"
    assert "token expired" in exc_info.value.message


def test_investor_flow_raises_on_http_error():
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(status_code=500)

    kis = _make_client(handler)
    with pytest.raises(httpx.HTTPStatusError):
        kis.fetch_daily_flow("005930")


def test_investor_flow_handles_space_padded_integers():
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            status_code=200,
            json={
                "rt_cd": "0",
                "output": [
                    {
                        "stck_bsop_date": "20260501",
                        "frgn_ntby_qty": "  1000  ",
                        "orgn_ntby_qty": "  ",
                        "frgn_ntby_tr_pbmn": "  5000000  ",
                        "orgn_ntby_tr_pbmn": "  ",
                    }
                ],
            },
        )

    kis = _make_client(handler)
    result = kis.fetch_daily_flow("005930")

    assert result == [
        DomesticInvestorFlow(
            date="20260501",
            foreign_net_qty=1000,
            institution_net_qty=0,
            individual_net_qty=0,
            foreign_net_krw=5000000,
            institution_net_krw=0,
            individual_net_krw=0,
        )
    ]


def test_investor_flow_refreshes_token_on_expiry():
    call_count = 0

    def handler(request: httpx.Request) -> httpx.Response:
        nonlocal call_count
        call_count += 1
        if call_count == 1:
            return httpx.Response(
                status_code=500,
                json={"rt_cd": "1", "msg_cd": "EGW00123", "msg1": "token expired"},
            )
        return httpx.Response(
            status_code=200,
            json={"rt_cd": "0", "output": []},
        )

    from unittest.mock import Mock

    token_manager = Mock()
    token_manager.get_token.return_value = "new-token"

    transport = httpx.MockTransport(handler)
    http = httpx.Client(
        transport=transport, base_url="https://openapi.koreainvestment.com:9443"
    )
    kis = KisDomesticInvestorClient(
        client=http,
        app_key="app-key",
        app_secret="app-secret",
        access_token="old-token",
        cust_type="P",
        env="real",
        token_manager=token_manager,
    )

    result = kis.fetch_daily_flow("005930")

    assert result == []
    assert call_count == 2
    assert token_manager.get_token.call_count == 1


def test_investor_flow_parses_individual_investor_fields():
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            status_code=200,
            json={
                "rt_cd": "0",
                "output": [
                    {
                        "stck_bsop_date": "20260503",
                        "frgn_ntby_qty": "100",
                        "orgn_ntby_qty": "200",
                        "prsn_ntby_qty": "300",
                        "frgn_ntby_tr_pbmn": "1000000",
                        "orgn_ntby_tr_pbmn": "2000000",
                        "prsn_ntby_tr_pbmn": "3000000",
                    }
                ],
            },
        )

    kis = _make_client(handler)
    result = kis.fetch_daily_flow("005930")

    assert result == [
        DomesticInvestorFlow(
            date="20260503",
            foreign_net_qty=100,
            institution_net_qty=200,
            individual_net_qty=300,
            foreign_net_krw=1000000,
            institution_net_krw=2000000,
            individual_net_krw=3000000,
        )
    ]
