from __future__ import annotations

from dataclasses import dataclass

import httpx

from portfolio_manager.services.kis.kis_base_client import KisBaseClient


@dataclass(frozen=True)
class DomesticStockInfo:
    pdno: str
    prdt_type_cd: str
    market_id: str
    name: str


@dataclass(frozen=True)
class KisDomesticInfoClient(KisBaseClient):
    client: httpx.Client
    app_key: str
    app_secret: str
    access_token: str
    tr_id: str
    cust_type: str

    def fetch_basic_info(self, prdt_type_cd: str, pdno: str) -> DomesticStockInfo:
        response = self.client.get(
            "/uapi/domestic-stock/v1/quotations/search-stock-info",
            params={
                "PRDT_TYPE_CD": prdt_type_cd,
                "PDNO": pdno,
            },
            headers=self._build_headers(self.tr_id),
        )
        response.raise_for_status()
        data = response.json()
        output = data.get("output")
        if isinstance(output, list):
            output = output[0] if output else None
        if not output:
            raise ValueError("KIS response missing output for stock info")
        name = (
            output.get("prdt_name")
            or output.get("prdt_name120")
            or output.get("prdt_abrv_name")
            or output.get("prdt_eng_name")
            or ""
        )
        return DomesticStockInfo(
            pdno=output["pdno"],
            prdt_type_cd=output["prdt_type_cd"],
            market_id=output["mket_id_cd"],
            name=name,
        )
