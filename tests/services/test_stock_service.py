"""Tests for StockService.resolve_and_persist_name."""

import logging
from datetime import datetime, timezone
from decimal import Decimal
from uuid import uuid4

from portfolio_manager.models import Stock
from portfolio_manager.services.stock_service import StockService


def _make_stock(name: str = "") -> Stock:
    now = datetime.now(timezone.utc)
    return Stock(
        id=uuid4(),
        ticker="005930",
        group_id=uuid4(),
        created_at=now,
        updated_at=now,
        exchange=None,
        name=name,
    )


class _FakeStockRepository:
    def __init__(self):
        self.update_name_calls: list[tuple] = []

    def update_name(self, stock_id, name) -> Stock:
        self.update_name_calls.append((stock_id, name))
        now = datetime.now(timezone.utc)
        return Stock(
            id=stock_id,
            ticker="",
            group_id=uuid4(),
            name=name,
            created_at=now,
            updated_at=now,
        )


class _FakePriceService:
    def __init__(self, name: str):
        self._name = name

    def get_stock_price(self, ticker, *, preferred_exchange=None):
        return (Decimal("70000"), "KRW", self._name, None)


class _ErrorPriceService:
    def get_stock_price(self, ticker, *, preferred_exchange=None):
        raise RuntimeError("network error")


def test_returns_formatted_existing_name_without_persisting():
    repo = _FakeStockRepository()
    service = StockService(repo, _FakePriceService("ignored"))
    stock = _make_stock("KODEX 200증권상장지수투자신탁(주식)")

    result = service.resolve_and_persist_name(stock)

    assert result == "KODEX 200"
    assert repo.update_name_calls == []


def test_resolves_etf_name_and_persists():
    repo = _FakeStockRepository()
    service = StockService(
        repo, _FakePriceService("KODEX 200증권상장지수투자신탁(주식)")
    )
    stock = _make_stock("")

    result = service.resolve_and_persist_name(stock)

    assert result == "KODEX 200"
    assert len(repo.update_name_calls) == 1
    assert repo.update_name_calls[0] == (stock.id, "KODEX 200")
    assert stock.name == "KODEX 200"


def test_exception_from_price_service_returns_empty(caplog):
    repo = _FakeStockRepository()
    service = StockService(repo, _ErrorPriceService())
    stock = _make_stock("")

    with caplog.at_level(logging.WARNING):
        result = service.resolve_and_persist_name(stock)

    assert result == ""
    assert repo.update_name_calls == []
    assert any("get_stock_price failed" in r.message for r in caplog.records)


def test_no_price_service_returns_empty():
    repo = _FakeStockRepository()
    service = StockService(repo, price_service=None)
    stock = _make_stock("")

    result = service.resolve_and_persist_name(stock)

    assert result == ""
    assert repo.update_name_calls == []


def test_price_service_returns_empty_name_skips_persist():
    repo = _FakeStockRepository()
    service = StockService(repo, _FakePriceService(""))
    stock = _make_stock("")

    result = service.resolve_and_persist_name(stock)

    assert result == ""
    assert repo.update_name_calls == []


def test_price_service_returns_suffix_only_name_skips_persist():
    repo = _FakeStockRepository()
    service = StockService(repo, _FakePriceService("증권상장지수투자신탁(주식)"))
    stock = _make_stock("")

    result = service.resolve_and_persist_name(stock)

    assert result == ""
    assert repo.update_name_calls == []


# --- persist_name ---


def test_persist_name_empty_stock_valid_raw_persists_and_returns_formatted():
    repo = _FakeStockRepository()
    service = StockService(repo)
    stock = _make_stock("")

    result = service.persist_name(stock, "KODEX 200증권상장지수투자신탁(주식)")

    assert result == "KODEX 200"
    assert repo.update_name_calls == [(stock.id, "KODEX 200")]
    assert stock.name == "KODEX 200"


def test_persist_name_stock_already_has_name_skips_persist_returns_formatted_raw():
    repo = _FakeStockRepository()
    service = StockService(repo)
    stock = _make_stock("삼성전자")

    result = service.persist_name(stock, "KODEX 200증권상장지수투자신탁(주식)")

    assert result == "KODEX 200"
    assert repo.update_name_calls == []
    assert stock.name == "삼성전자"


def test_persist_name_empty_raw_returns_empty_skips_persist():
    repo = _FakeStockRepository()
    service = StockService(repo)
    stock = _make_stock("")

    result = service.persist_name(stock, "")

    assert result == ""
    assert repo.update_name_calls == []


def test_persist_name_stock_has_name_empty_raw_returns_stored_name():
    repo = _FakeStockRepository()
    service = StockService(repo)
    stock = _make_stock("삼성전자")

    result = service.persist_name(stock, "")

    assert result == "삼성전자"
    assert repo.update_name_calls == []
    assert stock.name == "삼성전자"


def test_persist_name_suffix_only_raw_returns_empty_skips_persist():
    repo = _FakeStockRepository()
    service = StockService(repo)
    stock = _make_stock("")

    result = service.persist_name(stock, "증권상장지수투자신탁(주식)")

    assert result == ""
    assert repo.update_name_calls == []


# --- has_price_service ---


def test_has_price_service_false_when_none():
    service = StockService(_FakeStockRepository(), price_service=None)
    assert service.has_price_service is False


def test_has_price_service_true_when_set():
    service = StockService(_FakeStockRepository(), _FakePriceService("삼성전자"))
    assert service.has_price_service is True


# --- set_price_service ---


def test_set_price_service_enables_resolve():
    repo = _FakeStockRepository()
    service = StockService(repo, price_service=None)
    stock = _make_stock("")

    assert service.resolve_and_persist_name(stock) == ""

    service.set_price_service(_FakePriceService("KODEX 200증권상장지수투자신탁(주식)"))
    stock2 = _make_stock("")
    assert service.resolve_and_persist_name(stock2) == "KODEX 200"
    assert repo.update_name_calls == [(stock2.id, "KODEX 200")]
