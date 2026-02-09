from __future__ import annotations

from dataclasses import dataclass, replace

import httpx

from portfolio_manager.services.kis.kis_base_client import KisBaseClient
from portfolio_manager.services.kis.kis_error_handler import is_token_expired_error
from portfolio_manager.services.kis.kis_token_manager import TokenManager


@dataclass(frozen=True)
class KisDomesticOrderClient(KisBaseClient):
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
        cano: str,
        acnt_prdt_cd: str,
        pdno: str,
        ord_qty: str,
        ord_unpr: str,
        ord_dvsn: str = "00",
        excg_id_dvsn_cd: str = "KRX",
    ) -> dict:
        tr_id = self._resolve_tr_id(self.env, side=side)
        payload = {
            "CANO": cano,
            "ACNT_PRDT_CD": acnt_prdt_cd,
            "PDNO": pdno,
            "ORD_DVSN": ord_dvsn,
            "ORD_QTY": ord_qty,
            "ORD_UNPR": ord_unpr,
            "EXCG_ID_DVSN_CD": excg_id_dvsn_cd,
        }
        response = self.client.post(
            "/uapi/domestic-stock/v1/trading/order-cash",
            headers=self._build_headers(tr_id),
            json=payload,
        )

        if is_token_expired_error(response) and self.token_manager is not None:
            new_token = self.token_manager.get_token()
            refreshed = replace(self, access_token=new_token)
            response = refreshed.client.post(
                "/uapi/domestic-stock/v1/trading/order-cash",
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
                real_id="TTTC0012U",
                demo_id="VTTC0012U",
            )
        if side == "sell":
            return KisBaseClient._tr_id_for_env(
                env,
                real_id="TTTC0011U",
                demo_id="VTTC0011U",
            )
        raise ValueError("side must be one of: buy or sell")
