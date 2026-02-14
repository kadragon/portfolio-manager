from __future__ import annotations

from unittest.mock import Mock

import httpx

from portfolio_manager.services.kis.kis_base_client import KisBaseClient


class DummyClient(KisBaseClient):
    app_key = ""
    app_secret = ""
    access_token = ""
    cust_type = ""


def test_base_client_request_with_retry_uses_new_token_on_retry():
    client = DummyClient()
    client.access_token = "stale-token"
    expired_response = httpx.Response(
        status_code=500,
        json={"rt_cd": "1", "msg_cd": "EGW00123", "msg1": "expired"},
        request=httpx.Request("GET", "https://example.com"),
    )
    success_response = httpx.Response(
        status_code=200,
        json={"rt_cd": "0", "output": {}},
        request=httpx.Request("GET", "https://example.com"),
    )

    calls: list[str] = []

    def make_request(token_override: str | None) -> httpx.Response:
        token = token_override or client.access_token
        calls.append(token)
        if token == "stale-token":
            return expired_response
        return success_response

    token_manager = Mock()
    token_manager.get_token.return_value = "new-token"

    response = client._request_with_retry(make_request, token_manager=token_manager)

    assert response is success_response
    assert calls == ["stale-token", "new-token"]
    assert client.access_token == "new-token"

    calls.clear()
    response = client._request_with_retry(make_request, token_manager=token_manager)

    assert response is success_response
    assert calls == ["new-token"]
    assert token_manager.get_token.call_count == 1
