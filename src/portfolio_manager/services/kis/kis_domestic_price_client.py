from __future__ import annotations

from dataclasses import dataclass

import httpx

from portfolio_manager.services.kis.kis_base_client import KisBaseClient
from portfolio_manager.services.kis.kis_price_parser import PriceQuote


@dataclass(frozen=True)
class KisDomesticPriceClient(KisBaseClient):
    client: httpx.Client
    app_key: str
    app_secret: str
    access_token: str
    cust_type: str
    env: str

    def fetch_current_price(
        self, fid_cond_mrkt_div_code: str, fid_input_iscd: str
    ) -> PriceQuote:
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

    @staticmethod
    def _tr_id_for_env(
        env: str, *, real_id: str = "FHKST01010100", demo_id: str = "FHKST01010100"
    ) -> str:
        return KisBaseClient._tr_id_for_env(env, real_id=real_id, demo_id=demo_id)
