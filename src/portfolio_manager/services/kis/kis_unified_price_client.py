"""Unified KIS price client for both domestic and overseas stocks."""

import logging
from datetime import date

import httpx

from portfolio_manager.services.kis.kis_domestic_info_client import (
    KisDomesticInfoClient,
)
from portfolio_manager.services.kis.kis_domestic_price_client import (
    KisDomesticPriceClient,
)
from portfolio_manager.services.kis.kis_overseas_price_client import (
    KisOverseasPriceClient,
)
from portfolio_manager.services.kis.kis_market_detector import is_domestic_ticker
from portfolio_manager.services.kis.kis_price_parser import PriceQuote


class KisUnifiedPriceClient:
    """Unified price client that routes to domestic or overseas client."""

    @staticmethod
    def _get_prioritized_exchanges(preferred_exchange: str | None) -> list[str]:
        exchanges = ["NAS", "NYS", "AMS"]
        if preferred_exchange in exchanges:
            return [preferred_exchange] + [
                excd for excd in exchanges if excd != preferred_exchange
            ]
        return exchanges

    def __init__(
        self,
        domestic_client: KisDomesticPriceClient,
        overseas_client: KisOverseasPriceClient,
        domestic_info_client: KisDomesticInfoClient | None = None,
        prdt_type_cd: str = "300",
    ):
        """Initialize with domestic and overseas clients."""
        self.domestic_client = domestic_client
        self.overseas_client = overseas_client
        self.domestic_info_client = domestic_info_client
        self.prdt_type_cd = prdt_type_cd

    def get_price(
        self, ticker: str, preferred_exchange: str | None = None
    ) -> PriceQuote:
        """Get price for a ticker (auto-detects market)."""
        # Korean stocks are 6-character codes (e.g., "005930", "0052D0")
        if is_domestic_ticker(ticker):
            try:
                quote = self.domestic_client.fetch_current_price("J", ticker)
            except httpx.HTTPError as e:
                logging.warning(f"Failed to fetch domestic price for {ticker}: {e}")
                return PriceQuote(
                    symbol=ticker,
                    name="",
                    price=0.0,
                    market="KR",
                    currency="KRW",
                    exchange=None,
                )
            if quote.name or self.domestic_info_client is None:
                return quote
            try:
                info = self.domestic_info_client.fetch_basic_info(
                    prdt_type_cd=self.prdt_type_cd, pdno=ticker
                )
            except Exception:
                return quote
            return PriceQuote(
                symbol=quote.symbol,
                name=info.name,
                price=quote.price,
                market=quote.market,
                currency=quote.currency,
            )
        # US stocks are alphabetic symbols (e.g., "AAPL")
        else:
            exchanges = self._get_prioritized_exchanges(preferred_exchange)
            best_quote: PriceQuote | None = None
            for excd in exchanges:
                try:
                    quote = self.overseas_client.fetch_current_price(excd, ticker)
                except httpx.HTTPError as e:
                    logging.warning(
                        f"Failed to fetch overseas price for {ticker} from {excd}: {e}"
                    )
                    continue
                if best_quote is None:
                    best_quote = quote
                if quote.name:
                    return quote
                if best_quote.price == 0 and quote.price > 0:
                    best_quote = quote
                if preferred_exchange is not None:
                    return best_quote
            if best_quote is None:
                return PriceQuote(
                    symbol=ticker,
                    name="",
                    price=0.0,
                    market="US",
                    currency="USD",
                    exchange=None,
                )
            return best_quote

    def get_historical_close(
        self,
        ticker: str,
        target_date: date,
        preferred_exchange: str | None = None,
    ) -> float:
        """Get historical close price for a ticker (auto-detects market)."""
        if is_domestic_ticker(ticker):
            try:
                return float(
                    self.domestic_client.fetch_historical_close(
                        fid_input_iscd=ticker, target_date=target_date
                    )
                )
            except httpx.HTTPError as e:
                logging.warning(
                    f"Failed to fetch domestic historical close for {ticker}: {e}"
                )
                return 0.0
        exchanges = self._get_prioritized_exchanges(preferred_exchange)
        best_close = 0.0
        for excd in exchanges:
            try:
                close_price = self.overseas_client.fetch_historical_close(
                    excd=excd, symb=ticker, target_date=target_date
                )
            except httpx.HTTPError as e:
                logging.warning(
                    f"Failed to fetch overseas historical close for {ticker} from {excd}: {e}"
                )
                continue
            if close_price:
                return float(close_price)
            if best_close == 0.0 and close_price:
                best_close = float(close_price)
            if preferred_exchange is not None:
                return float(best_close)
        return float(best_close)
