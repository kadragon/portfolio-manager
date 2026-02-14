from __future__ import annotations

from dataclasses import dataclass

import httpx

from portfolio_manager.services.kis.kis_base_client import KisBaseClient
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

        def make_request(token_override: str | None) -> httpx.Response:
            headers = self._build_headers(tr_id)
            if token_override:
                headers["authorization"] = f"Bearer {token_override}"
            return self.client.post(
                "/uapi/domestic-stock/v1/trading/order-cash",
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
