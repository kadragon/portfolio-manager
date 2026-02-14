from __future__ import annotations

from dataclasses import dataclass
from datetime import date, timedelta

import httpx

from portfolio_manager.services.kis.kis_base_client import KisBaseClient
from portfolio_manager.services.kis.kis_price_parser import PriceQuote, parse_us_price
from portfolio_manager.services.kis.kis_token_manager import TokenManager

# Buffer days added to target_date when fetching historical prices
# to ensure the target date falls within the API's returned range
_DATE_FETCH_BUFFER_DAYS = 7


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
        return parse_us_price(data, symbol=symb, exchange=excd)

    def fetch_historical_close(
        self, excd: str, symb: str, target_date: date, auth: str = ""
    ) -> float:
        """Fetch historical close price for a given date.

        The KIS dailyprice API returns up to 100 trading days of data ending at BYMD.
        We set BYMD to target_date + buffer days to ensure target_date is within the range,
        then search the output2 array for the matching date.
        """
        tr_id = KisBaseClient._tr_id_for_env(
            self.env, real_id="HHDFS76240000", demo_id="HHDFS76240000"
        )
        # Set BYMD to after target_date to ensure target is in the returned range
        bymd = target_date + timedelta(days=_DATE_FETCH_BUFFER_DAYS)
        response = self.client.get(
            "/uapi/overseas-price/v1/quotations/dailyprice",
            params={
                "AUTH": auth,
                "EXCD": excd,
                "SYMB": symb,
                "GUBN": "0",
                "BYMD": bymd.strftime("%Y%m%d"),
                "MODP": "0",
            },
            headers=self._build_headers(tr_id),
        )
        response.raise_for_status()
        data = response.json()
        output2 = data.get("output2") or []
        if not isinstance(output2, list):
            output2 = [output2] if output2 else []

        target_str = target_date.strftime("%Y%m%d")
        for item in output2:
            if item.get("xymd") == target_str:
                raw_close = (item.get("clos") or "0").strip()
                return float(raw_close)

        # If exact date not found, return first available close as fallback
        if output2:
            raw_close = (output2[0].get("clos") or "0").strip()
            return float(raw_close)

        return 0.0

    def fetch_current_price_with_retry(
        self, excd: str, symb: str, token_manager: TokenManager, auth: str = ""
    ) -> PriceQuote:
        """Fetch current price with automatic token refresh on expiration."""
        tr_id = self._tr_id_for_env(self.env)
        params = {
            "AUTH": auth,
            "EXCD": excd,
            "SYMB": symb,
        }

        def make_request(token_override: str | None) -> httpx.Response:
            headers = self._build_headers(tr_id)
            if token_override:
                headers["authorization"] = f"Bearer {token_override}"
            return self.client.get(
                "/uapi/overseas-price/v1/quotations/price",
                params=params,
                headers=headers,
            )

        response = self._request_with_retry(make_request, token_manager=token_manager)
        data = response.json()
        return parse_us_price(data, symbol=symb, exchange=excd)

    @staticmethod
    def _tr_id_for_env(
        env: str, *, real_id: str = "HHDFS00000300", demo_id: str = "HHDFS00000300"
    ) -> str:
        return KisBaseClient._tr_id_for_env(env, real_id=real_id, demo_id=demo_id)
