import httpx
from datetime import date, timedelta

from portfolio_manager.services.kis.kis_domestic_price_client import (
    KisDomesticPriceClient,
)
from portfolio_manager.services.kis.kis_overseas_price_client import (
    KisOverseasPriceClient,
)


def test_historical_close_request_params_for_domestic_and_overseas():
    captured_domestic = {}

    def domestic_handler(request: httpx.Request) -> httpx.Response:
        captured_domestic["method"] = request.method
        captured_domestic["path"] = request.url.path
        captured_domestic["params"] = dict(request.url.params)
        captured_domestic["headers"] = dict(request.headers)
        return httpx.Response(
            status_code=200,
            json={"output2": [{"stck_clpr": "70000"}]},
        )

    domestic_client = httpx.Client(
        transport=httpx.MockTransport(domestic_handler),
        base_url="https://openapi.koreainvestment.com:9443",
    )
    domestic = KisDomesticPriceClient(
        client=domestic_client,
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        cust_type="P",
        env="real",
    )

    target_date = date(2024, 1, 15)
    close_price = domestic.fetch_historical_close(
        fid_input_iscd="005930", target_date=target_date
    )

    assert close_price == 70000
    assert captured_domestic["method"] == "GET"
    assert (
        captured_domestic["path"]
        == "/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice"
    )
    assert captured_domestic["params"] == {
        "FID_COND_MRKT_DIV_CODE": "J",
        "FID_INPUT_ISCD": "005930",
        "FID_INPUT_DATE_1": (target_date - timedelta(days=7)).strftime("%Y%m%d"),
        "FID_INPUT_DATE_2": target_date.strftime("%Y%m%d"),
        "FID_PERIOD_DIV_CODE": "D",
        "FID_ORG_ADJ_PRC": "1",
    }
    assert captured_domestic["headers"]["tr_id"] == "FHKST03010100"

    captured_overseas = {}

    def overseas_handler(request: httpx.Request) -> httpx.Response:
        captured_overseas["method"] = request.method
        captured_overseas["path"] = request.url.path
        captured_overseas["params"] = dict(request.url.params)
        captured_overseas["headers"] = dict(request.headers)
        return httpx.Response(
            status_code=200,
            json={"output2": [{"xymd": "20240115", "clos": "150.0"}]},
        )

    overseas_client = httpx.Client(
        transport=httpx.MockTransport(overseas_handler),
        base_url="https://openapi.koreainvestment.com:9443",
    )
    overseas = KisOverseasPriceClient(
        client=overseas_client,
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        cust_type="P",
        env="real",
    )

    overseas_close = overseas.fetch_historical_close(
        excd="NAS", symb="AAPL", target_date=date(2024, 1, 15)
    )

    assert overseas_close == 150.0
    assert captured_overseas["method"] == "GET"
    assert captured_overseas["path"] == "/uapi/overseas-price/v1/quotations/dailyprice"
    assert captured_overseas["params"] == {
        "AUTH": "",
        "EXCD": "NAS",
        "SYMB": "AAPL",
        "GUBN": "0",
        "BYMD": "20240122",  # target_date (20240115) + 7 days
        "MODP": "0",
    }
    assert captured_overseas["headers"]["tr_id"] == "HHDFS76240000"


def test_domestic_historical_close_falls_back_to_previous_business_day():
    captured_domestic = {}

    def domestic_handler(request: httpx.Request) -> httpx.Response:
        captured_domestic["params"] = dict(request.url.params)
        return httpx.Response(
            status_code=200,
            json={
                "output2": [
                    {"stck_bsop_date": "20231229", "stck_clpr": "71000"},
                    {"stck_bsop_date": "20231228", "stck_clpr": "70500"},
                ]
            },
        )

    domestic_client = httpx.Client(
        transport=httpx.MockTransport(domestic_handler),
        base_url="https://openapi.koreainvestment.com:9443",
    )
    domestic = KisDomesticPriceClient(
        client=domestic_client,
        app_key="app-key",
        app_secret="app-secret",
        access_token="access-token",
        cust_type="P",
        env="real",
    )

    close_price = domestic.fetch_historical_close(
        fid_input_iscd="005930", target_date=date(2024, 1, 1)
    )

    assert close_price == 71000
    assert captured_domestic["params"]["FID_INPUT_DATE_1"] == (
        date(2024, 1, 1) - timedelta(days=7)
    ).strftime("%Y%m%d")
    assert captured_domestic["params"]["FID_INPUT_DATE_2"] == "20240101"
