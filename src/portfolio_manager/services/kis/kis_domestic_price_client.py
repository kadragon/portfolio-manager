from __future__ import annotations

from dataclasses import dataclass
from datetime import date, timedelta

import httpx

from portfolio_manager.services.kis.kis_base_client import KisBaseClient
from portfolio_manager.services.kis.kis_price_parser import (
    PriceQuote,
    parse_korea_price,
)
from portfolio_manager.services.kis.kis_token_manager import TokenManager


_DATE_FETCH_BUFFER_DAYS = 7


@dataclass(frozen=True)
class KisDomesticPriceClient(KisBaseClient):
    client: httpx.Client
    app_key: str
    app_secret: str
    access_token: str
    cust_type: str
    env: str
    token_manager: TokenManager | None = None

    def fetch_current_price(
        self, fid_cond_mrkt_div_code: str, fid_input_iscd: str
    ) -> PriceQuote:
        # Use retry logic if token_manager is available
        if self.token_manager:
            return self.fetch_current_price_with_retry(
                fid_cond_mrkt_div_code, fid_input_iscd, self.token_manager
            )

        tr_id = self._tr_id_for_env(self.env)
        response = self.client.get(
            "/uapi/domestic-stock/v1/quotations/inquire-price",
            params={
                "FID_COND_MRKT_DIV_CODE": fid_cond_mrkt_div_code,
                "FID_INPUT_ISCD": fid_input_iscd,
            },
            headers=self._build_headers(tr_id),
        )
        response.raise_for_status()
        data = response.json()
        return parse_korea_price(data, symbol=fid_input_iscd)

    def fetch_historical_close(
        self,
        fid_input_iscd: str,
        target_date: date,
        fid_cond_mrkt_div_code: str = "J",
    ) -> int:
        """Fetch historical close price for a given date."""
        tr_id = KisBaseClient._tr_id_for_env(
            self.env, real_id="FHKST03010100", demo_id="FHKST03010100"
        )
        start_date = target_date - timedelta(days=_DATE_FETCH_BUFFER_DAYS)
        response = self.client.get(
            "/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice",
            params={
                "FID_COND_MRKT_DIV_CODE": fid_cond_mrkt_div_code,
                "FID_INPUT_ISCD": fid_input_iscd,
                "FID_INPUT_DATE_1": start_date.strftime("%Y%m%d"),
                "FID_INPUT_DATE_2": target_date.strftime("%Y%m%d"),
                "FID_PERIOD_DIV_CODE": "D",
                "FID_ORG_ADJ_PRC": "1",
            },
            headers=self._build_headers(tr_id),
        )
        response.raise_for_status()
        data = response.json()
        output = data.get("output2") or data.get("output") or []
        target_str = target_date.strftime("%Y%m%d")
        item: dict[str, str] = {}
        if isinstance(output, list):
            for candidate in output:
                if candidate.get("stck_bsop_date") == target_str:
                    item = candidate
                    break
            if not item and output:
                item = output[0]
        elif output:
            item = output
        raw_close = (item.get("stck_clpr") or item.get("stck_prpr") or "0").strip()
        return int(raw_close) if raw_close else 0

    def fetch_current_price_with_retry(
        self,
        fid_cond_mrkt_div_code: str,
        fid_input_iscd: str,
        token_manager: TokenManager,
    ) -> PriceQuote:
        """Fetch current price with automatic token refresh on expiration."""
        tr_id = self._tr_id_for_env(self.env)
        params = {
            "FID_COND_MRKT_DIV_CODE": fid_cond_mrkt_div_code,
            "FID_INPUT_ISCD": fid_input_iscd,
        }

        def make_request(token_override: str | None) -> httpx.Response:
            headers = self._build_headers(tr_id)
            if token_override:
                headers["authorization"] = f"Bearer {token_override}"
            return self.client.get(
                "/uapi/domestic-stock/v1/quotations/inquire-price",
                params=params,
                headers=headers,
            )

        response = self._request_with_retry(make_request, token_manager=token_manager)
        data = response.json()
        return parse_korea_price(data, symbol=fid_input_iscd)

    @staticmethod
    def _tr_id_for_env(
        env: str, *, real_id: str = "FHKST01010100", demo_id: str = "FHKST01010100"
    ) -> str:
        return KisBaseClient._tr_id_for_env(env, real_id=real_id, demo_id=demo_id)
