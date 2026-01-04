from __future__ import annotations

from dataclasses import dataclass

import httpx

from portfolio_manager.services.kis.kis_base_client import KisBaseClient
from portfolio_manager.services.kis.kis_error_handler import is_token_expired_error
from portfolio_manager.services.kis.kis_price_parser import PriceQuote
from portfolio_manager.services.kis.kis_token_manager import TokenManager


@dataclass(frozen=True)
class KisOverseasPriceClient(KisBaseClient):
    client: httpx.Client
    app_key: str
    app_secret: str
    access_token: str
    cust_type: str
    env: str
    token_manager: TokenManager | None = None

    def fetch_current_price(self, excd: str, symb: str, auth: str = "") -> PriceQuote:
        # Use retry logic if token_manager is available
        if self.token_manager:
            return self.fetch_current_price_with_retry(
                excd, symb, self.token_manager, auth
            )

        tr_id = self._tr_id_for_env(self.env)
        response = self.client.get(
            "/uapi/overseas-price/v1/quotations/price",
            params={
                "AUTH": auth,
                "EXCD": excd,
                "SYMB": symb,
            },
            headers=self._build_headers(tr_id),
        )
        response.raise_for_status()
        data = response.json()
        output = data["output"]
        if isinstance(output, list):
            output = output[0] if output else {}
        name = ""
        for key in (
            "name",
            "enname",
            "ename",
            "en_name",
            "symb_name",
            "symbol_name",
            "prdt_name",
            "product_name",
            "item_name",
        ):
            value = output.get(key)
            if isinstance(value, str) and value.strip():
                name = value.strip()
                break
        symbol = (
            output.get("symbol") or output.get("symb") or output.get("rsym") or symb
        )
        raw_last = (output.get("last") or "").strip()
        price = float(raw_last) if raw_last else 0.0
        return PriceQuote(
            symbol=symbol,
            name=name,
            price=float(price),
            market="US",
            currency="USD",
        )

    def fetch_current_price_with_retry(
        self, excd: str, symb: str, token_manager: TokenManager, auth: str = ""
    ) -> PriceQuote:
        """Fetch current price with automatic token refresh on expiration."""
        tr_id = self._tr_id_for_env(self.env)
        response = self.client.get(
            "/uapi/overseas-price/v1/quotations/price",
            params={
                "AUTH": auth,
                "EXCD": excd,
                "SYMB": symb,
            },
            headers=self._build_headers(tr_id),
        )

        # If token expired, refresh and retry
        if is_token_expired_error(response):
            new_token = token_manager.get_token()
            response = self.client.get(
                "/uapi/overseas-price/v1/quotations/price",
                params={
                    "AUTH": auth,
                    "EXCD": excd,
                    "SYMB": symb,
                },
                headers={
                    "content-type": "application/json",
                    "authorization": f"Bearer {new_token}",
                    "appkey": self.app_key,
                    "appsecret": self.app_secret,
                    "tr_id": tr_id,
                    "custtype": self.cust_type,
                },
            )

        response.raise_for_status()
        data = response.json()
        output = data["output"]
        if isinstance(output, list):
            output = output[0] if output else {}
        name = ""
        for key in (
            "name",
            "enname",
            "ename",
            "en_name",
            "symb_name",
            "symbol_name",
            "prdt_name",
            "product_name",
            "item_name",
        ):
            value = output.get(key)
            if isinstance(value, str) and value.strip():
                name = value.strip()
                break
        symbol = (
            output.get("symbol") or output.get("symb") or output.get("rsym") or symb
        )
        raw_last = (output.get("last") or "").strip()
        price = float(raw_last) if raw_last else 0.0
        return PriceQuote(
            symbol=symbol,
            name=name,
            price=float(price),
            market="US",
            currency="USD",
        )

    @staticmethod
    def _tr_id_for_env(
        env: str, *, real_id: str = "HHDFS00000300", demo_id: str = "HHDFS00000300"
    ) -> str:
        return KisBaseClient._tr_id_for_env(env, real_id=real_id, demo_id=demo_id)
