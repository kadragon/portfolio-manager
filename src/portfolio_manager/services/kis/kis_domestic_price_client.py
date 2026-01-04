from __future__ import annotations

from dataclasses import dataclass

import httpx

from portfolio_manager.services.kis.kis_price_parser import PriceQuote


@dataclass(frozen=True)
class KisDomesticPriceClient:
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
            headers={
                "content-type": "application/json",
                "authorization": f"Bearer {self.access_token}",
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
    def _tr_id_for_env(env: str) -> str:
        env_normalized = env.strip().lower()
        if "/" in env_normalized:
            env_normalized = env_normalized.split("/", 1)[0]
        if env_normalized in {"real", "prod"}:
            return "FHKST01010100"
        if env_normalized in {"demo", "vps", "paper"}:
            return "FHKST01010100"
        raise ValueError("env must be one of: real/prod or demo/vps/paper")
