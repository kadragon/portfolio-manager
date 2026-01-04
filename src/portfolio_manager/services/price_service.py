"""Service for fetching stock prices."""

from decimal import Decimal


class PriceService:
    """Service for fetching stock prices."""

    def __init__(self, price_client):
        """Initialize with a price client."""
        self.price_client = price_client

    def get_stock_price(self, ticker: str) -> tuple[Decimal, str]:
        """Get current price and currency for a stock ticker."""
        quote = self.price_client.get_price(ticker)
        return Decimal(str(quote.price)), quote.currency
