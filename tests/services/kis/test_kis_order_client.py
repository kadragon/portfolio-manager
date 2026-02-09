import json
from unittest.mock import Mock

import httpx

from portfolio_manager.services.kis.kis_domestic_order_client import (
    KisDomesticOrderClient,
)
from portfolio_manager.services.kis.kis_overseas_order_client import (
    KisOverseasOrderClient,
)


def test_domestic_order_buy_real_uses_tr_id_and_endpoint():
    captured: dict[str, str] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["path"] = request.url.path
        captured["tr_id"] = request.headers["tr_id"]
        return httpx.Response(
            status_code=200,
            json={"rt_cd": "0", "msg_cd": "", "msg1": ""},
        )

    client = httpx.Client(
        transport=httpx.MockTransport(handler),
        base_url="https://openapi.koreainvestment.com:9443",
    )
    order_client = KisDomesticOrderClient(
        client=client,
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        cust_type="P",
        env="real",
    )

    order_client.place_order(
        side="buy",
        cano="12345678",
        acnt_prdt_cd="01",
        pdno="005930",
        ord_qty="1",
        ord_unpr="70000",
    )

    assert captured["path"] == "/uapi/domestic-stock/v1/trading/order-cash"
    assert captured["tr_id"] == "TTTC0012U"


def test_domestic_order_sell_real_uses_tr_id():
    captured: dict[str, str] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["path"] = request.url.path
        captured["tr_id"] = request.headers["tr_id"]
        return httpx.Response(
            status_code=200,
            json={"rt_cd": "0", "msg_cd": "", "msg1": ""},
        )

    client = httpx.Client(
        transport=httpx.MockTransport(handler),
        base_url="https://openapi.koreainvestment.com:9443",
    )
    order_client = KisDomesticOrderClient(
        client=client,
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        cust_type="P",
        env="real",
    )

    order_client.place_order(
        side="sell",
        cano="12345678",
        acnt_prdt_cd="01",
        pdno="005930",
        ord_qty="1",
        ord_unpr="70000",
    )

    assert captured["path"] == "/uapi/domestic-stock/v1/trading/order-cash"
    assert captured["tr_id"] == "TTTC0011U"


def test_domestic_order_demo_uses_demo_tr_ids_for_buy_and_sell():
    captured_tr_ids: list[str] = []

    def handler(request: httpx.Request) -> httpx.Response:
        captured_tr_ids.append(request.headers["tr_id"])
        return httpx.Response(
            status_code=200,
            json={"rt_cd": "0", "msg_cd": "", "msg1": ""},
        )

    client = httpx.Client(
        transport=httpx.MockTransport(handler),
        base_url="https://openapi.koreainvestment.com:9443",
    )
    order_client = KisDomesticOrderClient(
        client=client,
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        cust_type="P",
        env="demo",
    )

    order_client.place_order(
        side="buy",
        cano="12345678",
        acnt_prdt_cd="01",
        pdno="005930",
        ord_qty="1",
        ord_unpr="70000",
    )
    order_client.place_order(
        side="sell",
        cano="12345678",
        acnt_prdt_cd="01",
        pdno="005930",
        ord_qty="1",
        ord_unpr="70000",
    )

    assert captured_tr_ids == ["VTTC0012U", "VTTC0011U"]


def test_domestic_order_sends_uppercase_body_keys():
    captured_body: dict[str, str] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        nonlocal captured_body
        captured_body = json.loads(request.content.decode("utf-8"))
        return httpx.Response(
            status_code=200,
            json={"rt_cd": "0", "msg_cd": "", "msg1": ""},
        )

    client = httpx.Client(
        transport=httpx.MockTransport(handler),
        base_url="https://openapi.koreainvestment.com:9443",
    )
    order_client = KisDomesticOrderClient(
        client=client,
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        cust_type="P",
        env="real",
    )

    order_client.place_order(
        side="buy",
        cano="12345678",
        acnt_prdt_cd="01",
        pdno="005930",
        ord_qty="3",
        ord_unpr="70100",
        ord_dvsn="00",
        excg_id_dvsn_cd="KRX",
    )

    assert captured_body == {
        "CANO": "12345678",
        "ACNT_PRDT_CD": "01",
        "PDNO": "005930",
        "ORD_DVSN": "00",
        "ORD_QTY": "3",
        "ORD_UNPR": "70100",
        "EXCG_ID_DVSN_CD": "KRX",
    }


def test_overseas_order_real_uses_tr_ids_for_buy_and_sell():
    captured_tr_ids: list[str] = []

    def handler(request: httpx.Request) -> httpx.Response:
        captured_tr_ids.append(request.headers["tr_id"])
        return httpx.Response(
            status_code=200,
            json={"rt_cd": "0", "msg_cd": "", "msg1": ""},
        )

    client = httpx.Client(
        transport=httpx.MockTransport(handler),
        base_url="https://openapi.koreainvestment.com:9443",
    )
    order_client = KisOverseasOrderClient(
        client=client,
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        cust_type="P",
        env="real",
    )

    order_client.place_order(
        side="buy",
        ovrs_excg_cd="NASD",
        pdno="AAPL",
        ord_qty="1",
        ovrs_ord_unpr="200.00",
    )
    order_client.place_order(
        side="sell",
        ovrs_excg_cd="NASD",
        pdno="AAPL",
        ord_qty="1",
        ovrs_ord_unpr="200.00",
    )

    assert captured_tr_ids == ["TTTT1002U", "TTTT1006U"]


def test_overseas_order_demo_uses_demo_tr_ids_for_buy_and_sell():
    captured_tr_ids: list[str] = []

    def handler(request: httpx.Request) -> httpx.Response:
        captured_tr_ids.append(request.headers["tr_id"])
        return httpx.Response(
            status_code=200,
            json={"rt_cd": "0", "msg_cd": "", "msg1": ""},
        )

    client = httpx.Client(
        transport=httpx.MockTransport(handler),
        base_url="https://openapi.koreainvestment.com:9443",
    )
    order_client = KisOverseasOrderClient(
        client=client,
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        cust_type="P",
        env="demo",
    )

    order_client.place_order(
        side="buy",
        ovrs_excg_cd="NASD",
        pdno="AAPL",
        ord_qty="1",
        ovrs_ord_unpr="200.00",
    )
    order_client.place_order(
        side="sell",
        ovrs_excg_cd="NASD",
        pdno="AAPL",
        ord_qty="1",
        ovrs_ord_unpr="200.00",
    )

    assert captured_tr_ids == ["VTTT1002U", "VTTT1006U"]


def test_overseas_order_sends_endpoint_and_required_body_keys():
    captured_path = ""
    captured_body: dict[str, str] = {}

    def handler(request: httpx.Request) -> httpx.Response:
        nonlocal captured_path, captured_body
        captured_path = request.url.path
        captured_body = json.loads(request.content.decode("utf-8"))
        return httpx.Response(
            status_code=200,
            json={"rt_cd": "0", "msg_cd": "", "msg1": ""},
        )

    client = httpx.Client(
        transport=httpx.MockTransport(handler),
        base_url="https://openapi.koreainvestment.com:9443",
    )
    order_client = KisOverseasOrderClient(
        client=client,
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        cust_type="P",
        env="real",
    )

    order_client.place_order(
        side="buy",
        ovrs_excg_cd="NASD",
        pdno="AAPL",
        ord_qty="2",
        ovrs_ord_unpr="201.50",
        ord_dvsn="00",
    )

    assert captured_path == "/uapi/overseas-stock/v1/trading/order"
    assert captured_body == {
        "OVRS_EXCG_CD": "NASD",
        "PDNO": "AAPL",
        "ORD_QTY": "2",
        "OVRS_ORD_UNPR": "201.50",
        "ORD_DVSN": "00",
    }


def test_order_clients_retry_once_on_token_expiration():
    expired_response = httpx.Response(
        status_code=500,
        json={"rt_cd": "1", "msg_cd": "EGW00123", "msg1": "expired"},
        request=httpx.Request("POST", "https://example.com"),
    )
    success_response = httpx.Response(
        status_code=200,
        json={"rt_cd": "0", "msg_cd": "", "msg1": ""},
        request=httpx.Request("POST", "https://example.com"),
    )

    domestic_http = Mock(spec=httpx.Client)
    domestic_http.post.side_effect = [expired_response, success_response]
    domestic_token_manager = Mock()
    domestic_token_manager.get_token.return_value = "new-domestic-token"
    domestic_client = KisDomesticOrderClient(
        client=domestic_http,
        app_key="app-key",
        app_secret="app-secret",
        access_token="old-domestic-token",
        cust_type="P",
        env="real",
        token_manager=domestic_token_manager,
    )

    domestic_client.place_order(
        side="buy",
        cano="12345678",
        acnt_prdt_cd="01",
        pdno="005930",
        ord_qty="1",
        ord_unpr="70000",
    )

    assert domestic_http.post.call_count == 2
    assert domestic_token_manager.get_token.call_count == 1
    assert (
        domestic_http.post.call_args_list[1].kwargs["headers"]["authorization"]
        == "Bearer new-domestic-token"
    )

    overseas_http = Mock(spec=httpx.Client)
    overseas_http.post.side_effect = [expired_response, success_response]
    overseas_token_manager = Mock()
    overseas_token_manager.get_token.return_value = "new-overseas-token"
    overseas_client = KisOverseasOrderClient(
        client=overseas_http,
        app_key="app-key",
        app_secret="app-secret",
        access_token="old-overseas-token",
        cust_type="P",
        env="real",
        token_manager=overseas_token_manager,
    )

    overseas_client.place_order(
        side="buy",
        ovrs_excg_cd="NASD",
        pdno="AAPL",
        ord_qty="1",
        ovrs_ord_unpr="200.00",
    )

    assert overseas_http.post.call_count == 2
    assert overseas_token_manager.get_token.call_count == 1
    assert (
        overseas_http.post.call_args_list[1].kwargs["headers"]["authorization"]
        == "Bearer new-overseas-token"
    )
