from __future__ import annotations

from dataclasses import dataclass

import httpx

from portfolio_manager.services.kis.kis_base_client import KisBaseClient

_EXCD_TO_PRDT_TYPE_CD: dict[str, str] = {
    "NAS": "512",
    "NYS": "513",
    "AMS": "529",
}


@dataclass(frozen=True)
class OverseasStockInfo:
    pdno: str
    prdt_type_cd: str
    excd: str
    name: str


@dataclass(frozen=True)
class KisOverseasInfoClient(KisBaseClient):
    client: httpx.Client
    app_key: str
    app_secret: str
    access_token: str
    tr_id: str
    cust_type: str

    def fetch_basic_info(self, excd: str, symb: str) -> OverseasStockInfo:
        prdt_type_cd = _EXCD_TO_PRDT_TYPE_CD.get(excd.upper(), "512")
        response = self.client.get(
            "/uapi/overseas-price/v1/quotations/search-info",
            params={
                "PRDT_TYPE_CD": prdt_type_cd,
                "PDNO": symb,
            },
            headers=self._build_headers(self.tr_id),
        )
        response.raise_for_status()
        data = response.json()
        if data.get("rt_cd") not in (None, "0"):
            raise ValueError(
                f"KIS overseas info error [{data.get('msg_cd')}]: {data.get('msg1')}"
            )
        output = data.get("output")
        if isinstance(output, list):
            output = output[0] if output else None
        if not output:
            raise ValueError(f"KIS response missing output for overseas info: {symb}")
        name = (
            output.get("prdt_name")
            or output.get("prdt_eng_name")
            or output.get("prdt_name120")
            or output.get("prdt_abrv_name")
            or ""
        )
        return OverseasStockInfo(
            pdno=output.get("pdno", symb),
            prdt_type_cd=prdt_type_cd,
            excd=excd,
            name=name,
        )
