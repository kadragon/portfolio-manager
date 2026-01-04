"""Service for FX rates used in portfolio valuation."""

from __future__ import annotations

from dataclasses import dataclass
from datetime import date, timedelta
from decimal import Decimal

from portfolio_manager.services.exim_exchange_rate_client import (
    EximExchangeRateClient,
)


@dataclass(frozen=True)
class ExchangeRateService:
    """Provide USD/KRW exchange rate from a fixed value or EXIM client."""

    exim_client: EximExchangeRateClient | None = None
    fixed_usd_krw_rate: Decimal | None = None

    def get_usd_krw_rate(self, search_date: str | None = None) -> Decimal:
        """Return USD/KRW rate as Decimal."""
        if self.fixed_usd_krw_rate is not None:
            return self.fixed_usd_krw_rate
        if self.exim_client is None:
            raise ValueError("Exchange rate source is not configured")
        if search_date is None:
            base_date = date.today()
            for offset in range(0, 7):
                candidate = (base_date - timedelta(days=offset)).strftime("%Y%m%d")
                try:
                    rate = self.exim_client.fetch_usd_rate(search_date=candidate)
                except ValueError as exc:
                    if str(exc) != "USD rate not found":
                        raise
                    continue
                return Decimal(str(rate))
            raise ValueError("USD rate not found")
        rate = self.exim_client.fetch_usd_rate(search_date=search_date)
        return Decimal(str(rate))
