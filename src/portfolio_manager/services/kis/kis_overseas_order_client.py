from __future__ import annotations

from dataclasses import dataclass, replace

import httpx

from portfolio_manager.services.kis.kis_base_client import KisBaseClient
from portfolio_manager.services.kis.kis_error_handler import is_token_expired_error
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
        response = self.client.post(
            "/uapi/overseas-stock/v1/trading/order",
            headers=self._build_headers(tr_id),
            json=payload,
        )

        if is_token_expired_error(response) and self.token_manager is not None:
            new_token = self.token_manager.get_token()
            refreshed = replace(self, access_token=new_token)
            response = refreshed.client.post(
                "/uapi/overseas-stock/v1/trading/order",
                headers=refreshed._build_headers(tr_id),
                json=payload,
            )

        response.raise_for_status()
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
