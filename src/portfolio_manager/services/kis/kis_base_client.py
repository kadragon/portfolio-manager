"""Shared behavior for KIS API clients."""

from __future__ import annotations

from typing import Callable

import httpx

from portfolio_manager.services.kis.kis_error_handler import is_token_expired_error
from portfolio_manager.services.kis.kis_token_manager import TokenManager


class KisBaseClient:
    """Base client with shared header and environment handling."""

    app_key: str
    app_secret: str
    access_token: str
    cust_type: str

    def _build_headers(self, tr_id: str) -> dict[str, str]:
        return {
            "content-type": "application/json",
            "authorization": f"Bearer {self.access_token}",
            "appkey": self.app_key,
            "appsecret": self.app_secret,
            "tr_id": tr_id,
            "custtype": self.cust_type,
        }

    def _request_with_retry(
        self,
        make_request: Callable[[str | None], httpx.Response],
        *,
        token_manager: TokenManager | None = None,
    ) -> httpx.Response:
        response = make_request(None)
        if is_token_expired_error(response) and token_manager is not None:
            new_token = token_manager.get_token()
            response = make_request(new_token)
        response.raise_for_status()
        return response

    @staticmethod
    def _tr_id_for_env(env: str, *, real_id: str, demo_id: str) -> str:
        env_normalized = env.strip().lower()
        if "/" in env_normalized:
            env_normalized = env_normalized.split("/", 1)[0]
        if env_normalized in {"real", "prod"}:
            return real_id
        if env_normalized in {"demo", "vps", "paper"}:
            return demo_id
        raise ValueError("env must be one of: real/prod or demo/vps/paper")
