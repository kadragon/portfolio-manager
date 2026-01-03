from __future__ import annotations

from dataclasses import dataclass

import httpx


@dataclass(frozen=True)
class EximExchangeRateClient:
    client: httpx.Client
    auth_key: str

    def fetch_usd_rate(self, search_date: str) -> float:
        response = self.client.get(
            "/site/program/financial/exchangeJSON",
            params={
                "authkey": self.auth_key,
                "searchdate": search_date,
                "data": "AP01",
            },
        )
        response.raise_for_status()
        data = response.json()
        for item in data:
            if item.get("cur_unit") == "USD":
                raw_rate = item.get("deal_bas_r", "")
                return float(raw_rate.replace(",", ""))
        raise ValueError("USD rate not found")
