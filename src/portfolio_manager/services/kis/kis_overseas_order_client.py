from __future__ import annotations

from dataclasses import dataclass

import httpx

from portfolio_manager.services.kis.kis_base_client import KisBaseClient
from portfolio_manager.services.kis.kis_token_manager import TokenManager


@dataclass(frozen=True)
class KisOverseasOrderClient(KisBaseClient):
    client: httpx.Client
    app_key: str
    app_secret: str
    access_token: str
    cust_type: str
    env: str
    token_manager: TokenManager | None = None

    def place_order(
        self,
        *,
        side: str,
        ovrs_excg_cd: str,
        pdno: str,
        ord_qty: str,
        ovrs_ord_unpr: str,
        ord_dvsn: str = "00",
    ) -> dict:
        tr_id = self._resolve_tr_id(self.env, side=side)
        payload = {
            "OVRS_EXCG_CD": ovrs_excg_cd,
            "PDNO": pdno,
            "ORD_QTY": ord_qty,
            "OVRS_ORD_UNPR": ovrs_ord_unpr,
            "ORD_DVSN": ord_dvsn,
        }

        def make_request(token_override: str | None) -> httpx.Response:
            headers = self._build_headers(tr_id)
            if token_override:
                headers["authorization"] = f"Bearer {token_override}"
            return self.client.post(
                "/uapi/overseas-stock/v1/trading/order",
                headers=headers,
                json=payload,
            )

        response = self._request_with_retry(
            make_request,
            token_manager=self.token_manager,
        )
        return response.json()

    @staticmethod
    def _resolve_tr_id(env: str, *, side: str) -> str:
        if side == "buy":
            return KisBaseClient._tr_id_for_env(
                env,
                real_id="TTTT1002U",
                demo_id="VTTT1002U",
            )
        if side == "sell":
            return KisBaseClient._tr_id_for_env(
                env,
                real_id="TTTT1006U",
                demo_id="VTTT1006U",
            )
        raise ValueError("side must be one of: buy or sell")
