"""Tests for KIS domestic account balance client."""

from decimal import Decimal
from unittest.mock import Mock

import httpx

from portfolio_manager.services.kis.kis_domestic_balance_client import (
    KisDomesticBalanceClient,
)


def test_fetch_account_snapshot_uses_expected_request_and_parses_response():
    captured_method = ""
    captured_path = ""
    captured_params: dict[str, str] = {}
    captured_headers: dict[str, str] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        nonlocal captured_method, captured_path, captured_params, captured_headers
        captured_method = request.method
        captured_path = request.url.path
        captured_params = dict(request.url.params)
        captured_headers = dict(request.headers)
        return httpx.Response(
            status_code=200,
            json={
                "output1": [
                    {"pdno": "005930", "hldg_qty": "10"},
                    {"pdno": "000660", "hldg_qty": "0"},
                ],
                "output2": [{"dnca_tot_amt": "1234500"}],
            },
        )

    client = httpx.Client(
        transport=httpx.MockTransport(handler),
        base_url="https://openapi.koreainvestment.com:9443",
    )
    kis_client = KisDomesticBalanceClient(
        client=client,
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        cust_type="P",
        env="real",
    )

    snapshot = kis_client.fetch_account_snapshot("12345678", "01")

    assert captured_method == "GET"
    assert captured_path == "/uapi/domestic-stock/v1/trading/inquire-balance"
    assert captured_params == {
        "CANO": "12345678",
        "ACNT_PRDT_CD": "01",
        "AFHR_FLPR_YN": "N",
        "OFL_YN": "",
        "INQR_DVSN": "01",
        "UNPR_DVSN": "01",
        "FUND_STTL_ICLD_YN": "N",
        "FNCG_AMT_AUTO_RDPT_YN": "N",
        "PRCS_DVSN": "00",
        "CTX_AREA_FK100": "",
        "CTX_AREA_NK100": "",
    }
    assert captured_headers["authorization"] == "Bearer access-token"
    assert captured_headers["tr_id"] == "TTTC8434R"

    assert snapshot.cash_balance == Decimal("1234500")
    assert len(snapshot.holdings) == 1
    assert snapshot.holdings[0].ticker == "005930"
    assert snapshot.holdings[0].quantity == Decimal("10")


def test_fetch_account_snapshot_reads_all_pages_when_tr_cont_indicates_more_data():
    call_count = 0
    requested_params: list[dict[str, str]] = []

    def handler(request: httpx.Request) -> httpx.Response:
        nonlocal call_count
        call_count += 1
        params = dict(request.url.params)
        requested_params.append(params)

        if call_count == 1:
            return httpx.Response(
                status_code=200,
                headers={"tr_cont": "M"},
                json={
                    "output1": [{"pdno": "005930", "hldg_qty": "10"}],
                    "output2": [{"dnca_tot_amt": "100000"}],
                    "ctx_area_fk100": "NEXT_FK",
                    "ctx_area_nk100": "NEXT_NK",
                },
            )

        return httpx.Response(
            status_code=200,
            json={
                "output1": [{"pdno": "000660", "hldg_qty": "3"}],
                "output2": [{"dnca_tot_amt": "100000"}],
            },
        )

    client = httpx.Client(
        transport=httpx.MockTransport(handler),
        base_url="https://openapi.koreainvestment.com:9443",
    )
    kis_client = KisDomesticBalanceClient(
        client=client,
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        cust_type="P",
        env="real",
    )

    snapshot = kis_client.fetch_account_snapshot("12345678", "01")

    assert call_count == 2
    assert requested_params[1]["CTX_AREA_FK100"] == "NEXT_FK"
    assert requested_params[1]["CTX_AREA_NK100"] == "NEXT_NK"
    assert len(snapshot.holdings) == 2
    assert snapshot.cash_balance == Decimal("100000")


def test_fetch_account_snapshot_retries_when_token_expired():
    request_headers: list[dict[str, str]] = []

    def handler(request: httpx.Request) -> httpx.Response:
        request_headers.append(dict(request.headers))
        if len(request_headers) == 1:
            return httpx.Response(
                status_code=500,
                json={"msg_cd": "EGW00123", "msg1": "Expired token"},
            )
        return httpx.Response(
            status_code=200,
            json={
                "output1": [{"pdno": "005930", "hldg_qty": "1"}],
                "output2": [{"dnca_tot_amt": "50000"}],
            },
        )

    client = httpx.Client(
        transport=httpx.MockTransport(handler),
        base_url="https://openapi.koreainvestment.com:9443",
    )
    token_manager = Mock()
    token_manager.get_token.return_value = "new-token"
    kis_client = KisDomesticBalanceClient(
        client=client,
        app_key="app-key",
        app_secret="app-secret",
        access_token="old-token",
        cust_type="P",
        env="real",
        token_manager=token_manager,
    )

    snapshot = kis_client.fetch_account_snapshot("12345678", "01")

    assert snapshot.cash_balance == Decimal("50000")
    assert request_headers[0]["authorization"] == "Bearer old-token"
    assert request_headers[1]["authorization"] == "Bearer new-token"
    token_manager.get_token.assert_called_once()
