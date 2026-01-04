"""Unified KIS price client for both domestic and overseas stocks."""

from portfolio_manager.services.kis_domestic_price_client import KisDomesticPriceClient
from portfolio_manager.services.kis_overseas_price_client import KisOverseasPriceClient
from portfolio_manager.services.kis_price_parser import PriceQuote


class KisUnifiedPriceClient:
    """Unified price client that routes to domestic or overseas client."""

    def __init__(
        self,
        domestic_client: KisDomesticPriceClient,
        overseas_client: KisOverseasPriceClient,
    ):
        """Initialize with domestic and overseas clients."""
        self.domestic_client = domestic_client
        self.overseas_client = overseas_client

    def get_price(self, ticker: str) -> PriceQuote:
        """Get price for a ticker (auto-detects market)."""
        # Korean stocks are 6-digit numbers (e.g., "005930")
        if ticker.isdigit() and len(ticker) == 6:
            return self.domestic_client.fetch_current_price("J", ticker)
        # US stocks are alphabetic symbols (e.g., "AAPL")
        else:
            return self.overseas_client.fetch_current_price("NAS", ticker)
