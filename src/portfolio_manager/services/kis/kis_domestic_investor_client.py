from __future__ import annotations

from dataclasses import dataclass

import httpx

from portfolio_manager.services.kis.kis_base_client import KisBaseClient


@dataclass(frozen=True)
class DomesticInvestorFlow:
    date: str
    foreign_net_qty: int
    institution_net_qty: int
    foreign_net_krw: int
    institution_net_krw: int


@dataclass(frozen=True)
class KisDomesticInvestorClient(KisBaseClient):
    client: httpx.Client
    app_key: str
    app_secret: str
    access_token: str
    tr_id: str
    cust_type: str

    def fetch_daily_flow(self, ticker: str) -> list[DomesticInvestorFlow]:
        response = self.client.get(
            "/uapi/domestic-stock/v1/quotations/inquire-investor",
            params={
                "FID_COND_MRKT_DIV_CODE": "J",
                "FID_INPUT_ISCD": ticker,
            },
            headers=self._build_headers(self.tr_id),
        )
        response.raise_for_status()
        data = response.json()
        output = data.get("output") or []
        return [
            DomesticInvestorFlow(
                date=row.get("stck_bsop_date", ""),
                foreign_net_qty=int(row.get("frgn_ntby_qty") or 0),
                institution_net_qty=int(row.get("orgn_ntby_qty") or 0),
                foreign_net_krw=int(row.get("frgn_ntby_tr_pbmn") or 0),
                institution_net_krw=int(row.get("orgn_ntby_tr_pbmn") or 0),
            )
            for row in output
        ]
