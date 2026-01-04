"""Tests for PriceQuote currency field."""

from portfolio_manager.services.kis_price_parser import PriceQuote


def test_price_quote_has_currency_field():
    """PriceQuote는 currency 필드를 가진다."""
    quote = PriceQuote(
        symbol="005930",
        name="삼성전자",
        price=70000,
        market="KR",
        currency="KRW",
    )

    assert quote.currency == "KRW"


def test_domestic_price_uses_krw():
    """국내 주식은 KRW 화폐를 사용한다."""
    quote = PriceQuote(
        symbol="005930",
        name="삼성전자",
        price=70000,
        market="KR",
        currency="KRW",
    )

    assert quote.currency == "KRW"


def test_overseas_price_uses_usd():
    """해외 주식은 USD 화폐를 사용한다."""
    quote = PriceQuote(
        symbol="AAPL",
        name="Apple Inc.",
        price=150.0,
        market="US",
        currency="USD",
    )

    assert quote.currency == "USD"
