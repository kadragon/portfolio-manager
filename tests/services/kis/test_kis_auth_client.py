import json
from datetime import datetime

import httpx

from portfolio_manager.services.kis_auth_client import KisAuthClient


def test_access_token_request_posts_client_credentials():
    captured = {}

    def handler(request: httpx.Request) -> httpx.Response:
        captured["method"] = request.method
        captured["path"] = request.url.path
        captured["body"] = json.loads(request.content.decode())
        return httpx.Response(
            status_code=200,
            json={
                "access_token": "test-token",
                "expires_in": 3600,
                "token_type": "Bearer",
            },
        )

    transport = httpx.MockTransport(handler)
    client = httpx.Client(
        transport=transport, base_url="https://openapi.koreainvestment.com:9443"
    )

    auth = KisAuthClient(client=client, app_key="app-key", app_secret="app-secret")
    token = auth.request_access_token()

    assert captured["method"] == "POST"
    assert captured["path"] == "/oauth2/tokenP"
    assert captured["body"]["grant_type"] == "client_credentials"
    assert captured["body"]["appkey"] == "app-key"
    assert captured["body"]["appsecret"] == "app-secret"

    assert token.token == "test-token"
    assert token.expires_at > datetime.now()


def test_access_token_parses_expired_at_string():
    def handler(request: httpx.Request) -> httpx.Response:
        return httpx.Response(
            status_code=200,
            json={
                "access_token": "test-token",
                "access_token_token_expired": "2026-01-03 12:34:56",
            },
        )

    transport = httpx.MockTransport(handler)
    client = httpx.Client(
        transport=transport, base_url="https://openapi.koreainvestment.com:9443"
    )

    auth = KisAuthClient(client=client, app_key="app-key", app_secret="app-secret")
    token = auth.request_access_token()

    assert token.token == "test-token"
    assert token.expires_at == datetime(2026, 1, 3, 12, 34, 56)
