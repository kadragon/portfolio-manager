"""Service for fetching stock prices."""

from calendar import monthrange
from datetime import date, timedelta
from decimal import Decimal


class PriceService:
    """Service for fetching stock prices."""

    def __init__(self, price_client):
        """Initialize with a price client."""
        self.price_client = price_client

    def get_stock_price(self, ticker: str) -> tuple[Decimal, str, str]:
        """Get current price, currency, and name for a stock ticker."""
        quote = self.price_client.get_price(ticker)
        return Decimal(str(quote.price)), quote.currency, quote.name

    def get_stock_change_rates(
        self, ticker: str, as_of: date | None = None
    ) -> dict[str, Decimal]:
        """Get 1Y/6M/1M change rates compared to historical close prices."""
        if as_of is None:
            as_of = date.today()

        def shift_years(base_date: date, years: int) -> date:
            target_year = base_date.year - years
            last_day = monthrange(target_year, base_date.month)[1]
            target_day = min(base_date.day, last_day)
            return date(target_year, base_date.month, target_day)

        def shift_months(base_date: date, months: int) -> date:
            target_year = base_date.year
            target_month = base_date.month - months
            while target_month <= 0:
                target_month += 12
                target_year -= 1
            last_day = monthrange(target_year, target_month)[1]
            target_day = min(base_date.day, last_day)
            return date(target_year, target_month, target_day)

        def adjust_to_previous_business_day(target_date: date) -> date:
            if target_date.weekday() == 5:
                return target_date - timedelta(days=1)
            if target_date.weekday() == 6:
                return target_date - timedelta(days=2)
            return target_date

        current_price, _, _ = self.get_stock_price(ticker)
        targets = {
            "1y": adjust_to_previous_business_day(shift_years(as_of, 1)),
            "6m": adjust_to_previous_business_day(shift_months(as_of, 6)),
            "1m": adjust_to_previous_business_day(shift_months(as_of, 1)),
        }
        change_rates: dict[str, Decimal] = {}
        for label, target_date in targets.items():
            past_close = Decimal(
                str(self.price_client.get_historical_close(ticker, target_date))
            )
            if past_close == 0:
                change_rates[label] = Decimal("0")
            else:
                change_rates[label] = (
                    (current_price - past_close) / past_close * Decimal("100")
                )
        return change_rates
