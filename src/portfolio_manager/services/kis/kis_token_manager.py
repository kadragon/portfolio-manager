from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime, timedelta

from portfolio_manager.services.auth_client import AuthClient
from portfolio_manager.services.kis.kis_token_store import TokenStore


@dataclass(frozen=True)
class TokenManager:
    store: TokenStore
    auth_client: AuthClient
    refresh_skew: timedelta = timedelta(minutes=1)

    def get_token(self) -> str:
        cached = self.store.load()
        now = datetime.now()
        if cached and cached.expires_at > now + self.refresh_skew:
            return cached.token

        new_token = self.auth_client.request_access_token()
        self.store.save(new_token.token, new_token.expires_at)
        return new_token.token
