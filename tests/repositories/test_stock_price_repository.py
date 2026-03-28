"""Tests for stock price repository."""

from datetime import date
from decimal import Decimal

from portfolio_manager.models.stock_price import StockPrice
from portfolio_manager.repositories.stock_price_repository import StockPriceRepository


def test_stock_price_repository_gets_by_ticker_and_date():
    repo = StockPriceRepository()
    repo.save(
        ticker="QQQ",
        price_date=date(2026, 1, 4),
        price=Decimal("410.12"),
        currency="USD",
        name="Invesco QQQ Trust",
        exchange="NAS",
    )

    result = repo.get_by_ticker_and_date("QQQ", date(2026, 1, 4))

    assert isinstance(result, StockPrice)
    assert result is not None
    assert result.price == Decimal("410.12")
    assert result.exchange == "NAS"


def test_stock_price_repository_get_returns_none_when_not_found():
    repo = StockPriceRepository()
    assert repo.get_by_ticker_and_date("XXX", date(2099, 1, 1)) is None


def test_stock_price_repository_saves_price():
    repo = StockPriceRepository()
    result = repo.save(
        ticker="QQQ",
        price_date=date(2026, 1, 4),
        price=Decimal("410.12"),
        currency="USD",
        name="Invesco QQQ Trust",
        exchange="",
    )

    assert result.id is not None
    assert result.price == Decimal("410.12")
    assert result.exchange == ""


def test_stock_price_repository_upserts_on_conflict():
    repo = StockPriceRepository()
    repo.save(
        ticker="QQQ",
        price_date=date(2026, 1, 4),
        price=Decimal("410.12"),
        currency="USD",
        name="Old Name",
        exchange="NAS",
    )
    updated = repo.save(
        ticker="QQQ",
        price_date=date(2026, 1, 4),
        price=Decimal("420.00"),
        currency="USD",
        name="New Name",
        exchange="NAS",
    )

    assert updated.price == Decimal("420.00")
    assert updated.name == "New Name"
