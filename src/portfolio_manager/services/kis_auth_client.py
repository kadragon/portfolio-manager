from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime, timedelta

import httpx

from portfolio_manager.services.kis_token_store import TokenData


@dataclass(frozen=True)
class KisAuthClient:
    client: httpx.Client
    app_key: str
    app_secret: str

    def request_access_token(self) -> TokenData:
        response = self.client.post(
            "/oauth2/tokenP",
            json={
                "grant_type": "client_credentials",
                "appkey": self.app_key,
                "appsecret": self.app_secret,
            },
        )
        response.raise_for_status()
        data = response.json()
        if "expires_in" in data:
            expires_in = int(data["expires_in"])
            expires_at = datetime.now() + timedelta(seconds=expires_in)
        else:
            expires_at = datetime.fromisoformat(data["access_token_token_expired"])
        return TokenData(token=data["access_token"], expires_at=expires_at)
