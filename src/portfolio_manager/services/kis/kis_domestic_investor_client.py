from __future__ import annotations

from dataclasses import dataclass

import httpx

from portfolio_manager.services.kis.kis_api_error import KisApiBusinessError
from portfolio_manager.services.kis.kis_base_client import KisBaseClient
from portfolio_manager.services.kis.kis_token_manager import TokenManager


def _parse_int(val: object) -> int:
    text = str(val).strip() if val is not None else ""
    return int(text) if text else 0


@dataclass(frozen=True)
class DomesticInvestorFlow:
    date: str
    foreign_net_qty: int
    institution_net_qty: int
    individual_net_qty: int
    foreign_net_krw: int
    institution_net_krw: int
    individual_net_krw: int


@dataclass(frozen=True)
class KisDomesticInvestorClient(KisBaseClient):
    client: httpx.Client
    app_key: str
    app_secret: str
    access_token: str
    cust_type: str
    env: str
    token_manager: TokenManager | None = None

    def fetch_daily_flow(self, ticker: str) -> list[DomesticInvestorFlow]:
        tr_id = self._tr_id_for_env(self.env)
        params = {
            "FID_COND_MRKT_DIV_CODE": "J",
            "FID_INPUT_ISCD": ticker,
        }

        def make_request(token_override: str | None) -> httpx.Response:
            headers = self._build_headers(tr_id)
            if token_override:
                headers["authorization"] = f"Bearer {token_override}"
            return self.client.get(
                "/uapi/domestic-stock/v1/quotations/inquire-investor",
                params=params,
                headers=headers,
            )

        response = self._request_with_retry(
            make_request, token_manager=self.token_manager
        )
        data = response.json()
        self._raise_for_business_error(data)
        output = data.get("output") or []
        return [
            DomesticInvestorFlow(
                date=row.get("stck_bsop_date", ""),
                foreign_net_qty=_parse_int(row.get("frgn_ntby_qty")),
                institution_net_qty=_parse_int(row.get("orgn_ntby_qty")),
                individual_net_qty=_parse_int(row.get("prsn_ntby_qty")),
                foreign_net_krw=_parse_int(row.get("frgn_ntby_tr_pbmn")),
                institution_net_krw=_parse_int(row.get("orgn_ntby_tr_pbmn")),
                individual_net_krw=_parse_int(row.get("prsn_ntby_tr_pbmn")),
            )
            for row in output
        ]

    @staticmethod
    def _raise_for_business_error(data: dict[str, object]) -> None:
        rt_cd = str(data.get("rt_cd", "")).strip()
        if rt_cd in {"", "0"}:
            return
        raise KisApiBusinessError(
            code=str(data.get("msg_cd", "")).strip(),
            message=str(data.get("msg1", "")).strip(),
        )

    @staticmethod
    def _tr_id_for_env(
        env: str, *, real_id: str = "FHKST01010900", demo_id: str = "FHKST01010900"
    ) -> str:
        return KisBaseClient._tr_id_for_env(env, real_id=real_id, demo_id=demo_id)
