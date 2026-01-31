"""Service for fetching stock prices."""

from calendar import monthrange
from datetime import date, timedelta
from decimal import Decimal, InvalidOperation

import httpx


class PriceService:
    """Service for fetching stock prices."""

    def __init__(
        self,
        price_client,
        price_cache_repository=None,
        today_provider=date.today,
    ):
        """Initialize with a price client."""
        self.price_client = price_client
        self.price_cache_repository = price_cache_repository
        self.today_provider = today_provider
        self._price_cache: dict[
            tuple[str, str | None], tuple[Decimal, str, str, str | None]
        ] = {}
        self._change_rates_cache: dict[
            tuple[str, date, str | None], dict[str, Decimal]
        ] = {}

    def get_stock_price(
        self, ticker: str, preferred_exchange: str | None = None
    ) -> tuple[Decimal, str, str, str | None]:
        """Get current price, currency, name, and exchange for a stock ticker."""
        # Check in-memory cache first (keyed by ticker and exchange)
        cache_key = (ticker, preferred_exchange)
        if cache_key in self._price_cache:
            return self._price_cache[cache_key]

        today = self.today_provider()
        cached = None
        if self.price_cache_repository:
            cached = self.price_cache_repository.get_by_ticker_and_date(ticker, today)
        if cached and cached.price > 0:
            result = (
                cached.price,
                cached.currency,
                cached.name,
                cached.exchange,
            )
            self._price_cache[cache_key] = result
            return result

        quote = self.price_client.get_price(
            ticker, preferred_exchange=preferred_exchange
        )
        price = Decimal(str(quote.price))
        if self.price_cache_repository and price > 0:
            self.price_cache_repository.save(
                ticker=ticker,
                price_date=today,
                price=price,
                currency=quote.currency,
                name=quote.name,
                exchange=quote.exchange,
            )
        result = (price, quote.currency, quote.name, quote.exchange)
        # Only cache valid prices (price > 0)
        if price > 0:
            self._price_cache[cache_key] = result
        return result

    def get_stock_change_rates(
        self,
        ticker: str,
        as_of: date | None = None,
        preferred_exchange: str | None = None,
    ) -> dict[str, Decimal]:
        """Get 1Y/6M/1M change rates compared to historical close prices."""
        if as_of is None:
            as_of = date.today()

        # Check in-memory cache first (keyed by ticker, date, and exchange)
        cache_key = (ticker, as_of, preferred_exchange)
        if cache_key in self._change_rates_cache:
            return self._change_rates_cache[cache_key]

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

        current_price, currency, name, exchange = self.get_stock_price(
            ticker, preferred_exchange=preferred_exchange
        )
        targets = {
            "1y": adjust_to_previous_business_day(shift_years(as_of, 1)),
            "6m": adjust_to_previous_business_day(shift_months(as_of, 6)),
            "1m": adjust_to_previous_business_day(shift_months(as_of, 1)),
        }
        change_rates: dict[str, Decimal] = {}
        for label, target_date in targets.items():
            cached = None
            if self.price_cache_repository:
                cached = self.price_cache_repository.get_by_ticker_and_date(
                    ticker, target_date
                )
            if cached and cached.price > 0:
                past_close = cached.price
            else:
                try:
                    past_close = Decimal(
                        str(
                            self.price_client.get_historical_close(
                                ticker,
                                target_date,
                                preferred_exchange=preferred_exchange,
                            )
                        )
                    )
                except (httpx.HTTPStatusError, InvalidOperation, ValueError, TypeError):
                    past_close = Decimal("0")
                if self.price_cache_repository and past_close > 0:
                    self.price_cache_repository.save(
                        ticker=ticker,
                        price_date=target_date,
                        price=past_close,
                        currency=currency,
                        name=name,
                        exchange=exchange,
                    )
            if past_close == 0:
                change_rates[label] = Decimal("0")
            else:
                change_rates[label] = (
                    (current_price - past_close) / past_close * Decimal("100")
                )
        self._change_rates_cache[cache_key] = change_rates
        return change_rates
