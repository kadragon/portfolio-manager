from __future__ import annotations

from dataclasses import dataclass

import httpx

from portfolio_manager.services.kis_price_parser import PriceQuote


@dataclass(frozen=True)
class KisOverseasPriceClient:
    client: httpx.Client
    app_key: str
    app_secret: str
    access_token: str
    cust_type: str
    env: str

    def fetch_current_price(self, excd: str, symb: str, auth: str = "") -> PriceQuote:
        tr_id = self._tr_id_for_env(self.env)
        response = self.client.get(
            "/uapi/overseas-price/v1/quotations/price",
            params={
                "AUTH": auth,
                "EXCD": excd,
                "SYMB": symb,
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
        if isinstance(output, list):
            output = output[0] if output else {}
        name = output.get("name") or ""
        symbol = (
            output.get("symbol") or output.get("symb") or output.get("rsym") or symb
        )
        price = float(output["last"])
        return PriceQuote(
            symbol=symbol,
            name=name,
            price=float(price),
            market="US",
        )

    @staticmethod
    def _tr_id_for_env(env: str) -> str:
        env_normalized = env.strip().lower()
        if "/" in env_normalized:
            env_normalized = env_normalized.split("/", 1)[0]
        if env_normalized in {"real", "prod"}:
            return "HHDFS00000300"
        if env_normalized in {"demo", "vps", "paper"}:
            return "HHDFS00000300"
        raise ValueError("env must be one of: real/prod or demo/vps/paper")
