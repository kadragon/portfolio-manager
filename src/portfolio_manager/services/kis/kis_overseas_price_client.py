from __future__ import annotations

from dataclasses import dataclass

import httpx

from portfolio_manager.services.kis.kis_base_client import KisBaseClient
from portfolio_manager.services.kis.kis_price_parser import PriceQuote


@dataclass(frozen=True)
class KisOverseasPriceClient(KisBaseClient):
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

    @staticmethod
    def _tr_id_for_env(
        env: str, *, real_id: str = "HHDFS00000300", demo_id: str = "HHDFS00000300"
    ) -> str:
        return KisBaseClient._tr_id_for_env(env, real_id=real_id, demo_id=demo_id)
