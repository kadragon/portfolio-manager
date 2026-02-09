"""Unified order client — routes to domestic or overseas KIS order API."""

from __future__ import annotations

from typing import Any

from portfolio_manager.services.kis.kis_market_detector import is_domestic_ticker


class KisUnifiedOrderClient:
    """Routes order requests to domestic or overseas KIS order clients."""

    def __init__(
        self,
        domestic_client: Any,
        overseas_client: Any,
        cano: str,
        acnt_prdt_cd: str,
        price_service: Any,
    ):
        self._domestic = domestic_client
        self._overseas = overseas_client
        self._cano = cano
        self._acnt_prdt_cd = acnt_prdt_cd
        self._price_service = price_service

    def place_order(
        self,
        *,
        ticker: str,
        side: str,
        quantity: int,
        exchange: str | None = None,
    ) -> dict:
        """Place an order, routing to domestic or overseas client."""
        price, _currency, _name, _exch = self._price_service.get_stock_price(
            ticker, preferred_exchange=exchange
        )

        if is_domestic_ticker(ticker):
            return self._domestic.place_order(
                side=side,
                cano=self._cano,
                acnt_prdt_cd=self._acnt_prdt_cd,
                pdno=ticker,
                ord_qty=str(quantity),
                ord_unpr=str(int(price)),
            )
        else:
            return self._overseas.place_order(
                side=side,
                ovrs_excg_cd=exchange or "NASD",
                pdno=ticker,
                ord_qty=str(quantity),
                ovrs_ord_unpr=str(price),
            )
