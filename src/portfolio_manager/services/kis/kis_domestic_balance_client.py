from __future__ import annotations

from dataclasses import dataclass
from decimal import Decimal, InvalidOperation
from collections import defaultdict

import httpx

from portfolio_manager.services.kis.kis_base_client import KisBaseClient
from portfolio_manager.services.kis.kis_error_handler import is_token_expired_error
from portfolio_manager.services.kis.kis_token_manager import TokenManager


@dataclass(frozen=True)
class KisHoldingPosition:
    """Single holding position from KIS account balance."""

    ticker: str
    quantity: Decimal


@dataclass(frozen=True)
class KisAccountSnapshot:
    """KIS account snapshot including cash and holdings."""

    cash_balance: Decimal
    holdings: list[KisHoldingPosition]


@dataclass(frozen=True)
class KisDomesticBalanceClient(KisBaseClient):
    """Client for KIS domestic account balance API."""

    client: httpx.Client
    app_key: str
    app_secret: str
    access_token: str
    cust_type: str
    env: str
    token_manager: TokenManager | None = None

    def fetch_account_snapshot(
        self, cano: str, acnt_prdt_cd: str
    ) -> KisAccountSnapshot:
        """Fetch cash balance and holdings for an account."""
        fk100 = ""
        nk100 = ""
        tr_cont = ""
        cash_balance = Decimal("0")
        holding_quantities: dict[str, Decimal] = defaultdict(Decimal)

        while True:
            response = self._request_balance(
                cano=cano,
                acnt_prdt_cd=acnt_prdt_cd,
                fk100=fk100,
                nk100=nk100,
                tr_cont=tr_cont,
            )
            data = response.json()
            output1 = data.get("output1") or []
            output2 = data.get("output2") or []

            if isinstance(output1, dict):
                output1 = [output1]
            if isinstance(output2, dict):
                output2 = [output2]

            for item in output1:
                ticker = str(item.get("pdno", "")).strip()
                quantity = self._parse_decimal(item.get("hldg_qty", "0"))
                if ticker and quantity > 0:
                    holding_quantities[ticker] += quantity

            if output2:
                cash_balance = self._parse_cash_balance(output2[0])

            header_tr_cont = response.headers.get("tr_cont", "")
            if header_tr_cont not in {"M", "F"}:
                break

            fk100 = str(data.get("ctx_area_fk100", "")).strip()
            nk100 = str(data.get("ctx_area_nk100", "")).strip()
            tr_cont = "N"

        holdings = [
            KisHoldingPosition(ticker=ticker, quantity=quantity)
            for ticker, quantity in sorted(holding_quantities.items())
        ]
        return KisAccountSnapshot(cash_balance=cash_balance, holdings=holdings)

    def _request_balance(
        self,
        *,
        cano: str,
        acnt_prdt_cd: str,
        fk100: str,
        nk100: str,
        tr_cont: str,
    ) -> httpx.Response:
        response = self.client.get(
            "/uapi/domestic-stock/v1/trading/inquire-balance",
            params={
                "CANO": cano,
                "ACNT_PRDT_CD": acnt_prdt_cd,
                "AFHR_FLPR_YN": "N",
                "OFL_YN": "",
                "INQR_DVSN": "01",
                "UNPR_DVSN": "01",
                "FUND_STTL_ICLD_YN": "N",
                "FNCG_AMT_AUTO_RDPT_YN": "N",
                "PRCS_DVSN": "00",
                "CTX_AREA_FK100": fk100,
                "CTX_AREA_NK100": nk100,
            },
            headers=self._build_headers_with_tr_cont(
                tr_id=self._tr_id_for_env(self.env),
                tr_cont=tr_cont,
            ),
        )

        if is_token_expired_error(response) and self.token_manager is not None:
            new_token = self.token_manager.get_token()
            response = self.client.get(
                "/uapi/domestic-stock/v1/trading/inquire-balance",
                params={
                    "CANO": cano,
                    "ACNT_PRDT_CD": acnt_prdt_cd,
                    "AFHR_FLPR_YN": "N",
                    "OFL_YN": "",
                    "INQR_DVSN": "01",
                    "UNPR_DVSN": "01",
                    "FUND_STTL_ICLD_YN": "N",
                    "FNCG_AMT_AUTO_RDPT_YN": "N",
                    "PRCS_DVSN": "00",
                    "CTX_AREA_FK100": fk100,
                    "CTX_AREA_NK100": nk100,
                },
                headers=self._build_headers_with_tr_cont(
                    tr_id=self._tr_id_for_env(self.env),
                    tr_cont=tr_cont,
                    access_token=new_token,
                ),
            )

        response.raise_for_status()
        return response

    def _build_headers_with_tr_cont(
        self, *, tr_id: str, tr_cont: str, access_token: str | None = None
    ) -> dict[str, str]:
        token = access_token or self.access_token
        headers = {
            "content-type": "application/json",
            "authorization": f"Bearer {token}",
            "appkey": self.app_key,
            "appsecret": self.app_secret,
            "tr_id": tr_id,
            "custtype": self.cust_type,
        }
        if tr_cont:
            headers["tr_cont"] = tr_cont
        return headers

    @staticmethod
    def _parse_decimal(value: object) -> Decimal:
        text = str(value).strip()
        if not text:
            return Decimal("0")
        try:
            return Decimal(text)
        except (InvalidOperation, ValueError):
            return Decimal("0")

    @classmethod
    def _parse_cash_balance(cls, output2_row: dict) -> Decimal:
        for key in ("dnca_tot_amt", "ord_psbl_cash", "tot_dnca_amt"):
            if key in output2_row:
                return cls._parse_decimal(output2_row.get(key))
        return Decimal("0")

    @staticmethod
    def _tr_id_for_env(
        env: str, *, real_id: str = "TTTC8434R", demo_id: str = "VTTC8434R"
    ) -> str:
        return KisBaseClient._tr_id_for_env(env, real_id=real_id, demo_id=demo_id)
