from __future__ import annotations

from dataclasses import dataclass
from datetime import date

import httpx

from portfolio_manager.services.kis.kis_base_client import KisBaseClient
from portfolio_manager.services.kis.kis_error_handler import is_token_expired_error
from portfolio_manager.services.kis.kis_price_parser import PriceQuote
from portfolio_manager.services.kis.kis_token_manager import TokenManager


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
        output = data["output"]
        name = output.get("hts_kor_isnm", "")
        price = int(output["stck_prpr"])
        return PriceQuote(
            symbol=fid_input_iscd,
            name=name,
            price=price,
            market="KR",
            currency="KRW",
        )

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
        response = self.client.get(
            "/uapi/domestic-stock/v1/quotations/inquire-daily-itemchartprice",
            params={
                "FID_COND_MRKT_DIV_CODE": fid_cond_mrkt_div_code,
                "FID_INPUT_ISCD": fid_input_iscd,
                "FID_INPUT_DATE_1": target_date.strftime("%Y%m%d"),
                "FID_INPUT_DATE_2": target_date.strftime("%Y%m%d"),
                "FID_PERIOD_DIV_CODE": "D",
                "FID_ORG_ADJ_PRC": "1",
            },
            headers=self._build_headers(tr_id),
        )
        response.raise_for_status()
        data = response.json()
        output = data.get("output2") or data.get("output") or []
        if isinstance(output, list):
            item = output[0] if output else {}
        else:
            item = output or {}
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
        response = self.client.get(
            "/uapi/domestic-stock/v1/quotations/inquire-price",
            params={
                "FID_COND_MRKT_DIV_CODE": fid_cond_mrkt_div_code,
                "FID_INPUT_ISCD": fid_input_iscd,
            },
            headers=self._build_headers(tr_id),
        )

        # If token expired, refresh and retry
        if is_token_expired_error(response):
            new_token = token_manager.get_token()
            response = self.client.get(
                "/uapi/domestic-stock/v1/quotations/inquire-price",
                params={
                    "FID_COND_MRKT_DIV_CODE": fid_cond_mrkt_div_code,
                    "FID_INPUT_ISCD": fid_input_iscd,
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
        name = output.get("hts_kor_isnm", "")
        price = int(output["stck_prpr"])
        return PriceQuote(
            symbol=fid_input_iscd,
            name=name,
            price=price,
            market="KR",
            currency="KRW",
        )

    @staticmethod
    def _tr_id_for_env(
        env: str, *, real_id: str = "FHKST01010100", demo_id: str = "FHKST01010100"
    ) -> str:
        return KisBaseClient._tr_id_for_env(env, real_id=real_id, demo_id=demo_id)
