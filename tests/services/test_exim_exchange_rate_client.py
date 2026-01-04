import httpx

from portfolio_manager.services.exchange.exim_exchange_rate_client import (
    EximExchangeRateClient,
)


def test_exim_exchange_rate_client_fetches_usd_rate_for_date():
    captured = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["path"] = request.url.path
        captured["params"] = dict(request.url.params)
        return httpx.Response(
            status_code=200,
            json=[
                {"cur_unit": "JPY(100)", "deal_bas_r": "951.05"},
                {"cur_unit": "USD", "deal_bas_r": "1,066.9"},
            ],
        )

    transport = httpx.MockTransport(handler)
    client = httpx.Client(transport=transport, base_url="https://oapi.koreaexim.go.kr")

    exim = EximExchangeRateClient(client=client, auth_key="auth-key")

    result = exim.fetch_usd_rate(search_date="20180102")

    assert captured["path"] == "/site/program/financial/exchangeJSON"
    assert captured["params"] == {
        "authkey": "auth-key",
        "searchdate": "20180102",
        "data": "AP01",
    }
    assert result == 1066.9
