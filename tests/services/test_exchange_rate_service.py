from datetime import date
from decimal import Decimal
from unittest.mock import Mock, call

import portfolio_manager.services.exchange.exchange_rate_service as exchange_rate_service
from portfolio_manager.services.exchange.exchange_rate_service import (
    ExchangeRateService,
)


def test_exchange_rate_service_falls_back_to_previous_day_when_rate_missing(
    monkeypatch,
):
    class FakeDate(date):
        @classmethod
        def today(cls):
            return cls(2026, 1, 4)

    monkeypatch.setattr(exchange_rate_service, "date", FakeDate)

    exim_client = Mock()

    def side_effect(search_date: str):
        if search_date in {"20260104", "20260103"}:
            raise ValueError("USD rate not found")
        if search_date == "20260102":
            return 1330.5
        raise AssertionError(f"unexpected search_date: {search_date}")

    exim_client.fetch_usd_rate.side_effect = side_effect

    service = ExchangeRateService(exim_client=exim_client)

    rate = service.get_usd_krw_rate()

    assert rate == Decimal("1330.5")
    assert exim_client.fetch_usd_rate.call_args_list == [
        call(search_date="20260104"),
        call(search_date="20260103"),
        call(search_date="20260102"),
    ]
