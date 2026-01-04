"""Unified KIS price client for both domestic and overseas stocks."""

from portfolio_manager.services.kis_domestic_info_client import KisDomesticInfoClient
from portfolio_manager.services.kis_domestic_price_client import KisDomesticPriceClient
from portfolio_manager.services.kis_overseas_price_client import KisOverseasPriceClient
from portfolio_manager.services.kis_price_parser import PriceQuote


class KisUnifiedPriceClient:
    """Unified price client that routes to domestic or overseas client."""

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

    def get_price(self, ticker: str) -> PriceQuote:
        """Get price for a ticker (auto-detects market)."""
        # Korean stocks are 6-character codes (e.g., "005930", "0052D0")
        if len(ticker) == 6:
            quote = self.domestic_client.fetch_current_price("J", ticker)
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
            return self.overseas_client.fetch_current_price("NAS", ticker)
