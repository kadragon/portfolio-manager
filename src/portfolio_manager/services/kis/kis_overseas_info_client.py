from __future__ import annotations

from dataclasses import dataclass

import httpx

from portfolio_manager.services.kis.kis_base_client import KisBaseClient


@dataclass(frozen=True)
class OverseasStockInfo:
    symbol: str
    exchange: str
    name: str


@dataclass(frozen=True)
class KisOverseasInfoClient(KisBaseClient):
    """Fetches overseas stock basic info from KIS.

    Uses /uapi/overseas-price/v1/quotations/search-overseas-info.
    tr_id and name field order must be verified against KIS API docs before
    enabling in production (see KIS_OVERSEAS_INFO_TR_ID env var).
    """

    client: httpx.Client
    app_key: str
    app_secret: str
    access_token: str
    tr_id: str
    cust_type: str

    def fetch_basic_info(self, excd: str, symb: str) -> OverseasStockInfo:
        response = self.client.get(
            "/uapi/overseas-price/v1/quotations/search-overseas-info",
            params={
                "AUTH": "",
                "EXCD": excd,
                "SYMB": symb,
            },
            headers=self._build_headers(self.tr_id),
        )
        response.raise_for_status()
        data = response.json()
        output = data.get("output")
        if isinstance(output, list):
            output = output[0] if output else None
        if not output:
            raise ValueError(f"KIS overseas info missing output for {symb}")
        name = (
            output.get("prdt_eng_name")
            or output.get("prdt_name")
            or output.get("prdt_name120")
            or output.get("prdt_abrv_name")
            or output.get("symb_name")
            or output.get("security_name")
            or ""
        )
        return OverseasStockInfo(
            symbol=output.get("symb") or output.get("rsym") or symb,
            exchange=excd,
            name=name.strip(),
        )
