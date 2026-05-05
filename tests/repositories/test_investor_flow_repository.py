"""Tests for InvestorFlowRepository."""

from datetime import date

from portfolio_manager.repositories.investor_flow_repository import (
    InvestorFlowRepository,
)
from portfolio_manager.models.investor_flow import InvestorFlow


def _save(repo: InvestorFlowRepository, ticker: str, flow_date: date, **overrides):
    defaults = dict(
        foreign_net_qty=100,
        institution_net_qty=200,
        individual_net_qty=300,
        foreign_net_krw=1_000_000,
        institution_net_krw=2_000_000,
        individual_net_krw=3_000_000,
    )
    defaults.update(overrides)
    return repo.save(ticker=ticker, flow_date=flow_date, **defaults)


def test_round_trip():
    repo = InvestorFlowRepository()
    saved = _save(repo, "005930", date(2026, 5, 1))

    result = repo.get_by_ticker_and_date("005930", date(2026, 5, 1))

    assert isinstance(result, InvestorFlow)
    assert result.ticker == "005930"
    assert result.flow_date == date(2026, 5, 1)
    assert result.foreign_net_qty == 100
    assert result.institution_net_qty == 200
    assert result.individual_net_qty == 300
    assert result.foreign_net_krw == 1_000_000
    assert result.institution_net_krw == 2_000_000
    assert result.individual_net_krw == 3_000_000
    assert result.id == saved.id


def test_save_upserts_on_duplicate():
    repo = InvestorFlowRepository()
    _save(repo, "005930", date(2026, 5, 1), foreign_net_qty=100)
    updated = _save(repo, "005930", date(2026, 5, 1), foreign_net_qty=999)

    result = repo.get_by_ticker_and_date("005930", date(2026, 5, 1))
    assert result is not None
    assert result.foreign_net_qty == 999
    assert result.id == updated.id


def test_get_returns_none_for_missing():
    repo = InvestorFlowRepository()
    assert repo.get_by_ticker_and_date("MISSING", date(2026, 1, 1)) is None


def test_list_by_ticker_range_returns_asc_order():
    repo = InvestorFlowRepository()
    _save(repo, "005930", date(2026, 5, 3))
    _save(repo, "005930", date(2026, 5, 1))
    _save(repo, "005930", date(2026, 5, 2))

    results = repo.list_by_ticker_range("005930", date(2026, 5, 1), date(2026, 5, 3))

    assert [r.flow_date for r in results] == [
        date(2026, 5, 1),
        date(2026, 5, 2),
        date(2026, 5, 3),
    ]


def test_list_by_ticker_range_excludes_other_tickers():
    repo = InvestorFlowRepository()
    _save(repo, "005930", date(2026, 5, 1))
    _save(repo, "000660", date(2026, 5, 1))

    results = repo.list_by_ticker_range("005930", date(2026, 5, 1), date(2026, 5, 1))

    assert len(results) == 1
    assert results[0].ticker == "005930"


def test_list_by_ticker_range_returns_empty_when_no_data():
    repo = InvestorFlowRepository()
    results = repo.list_by_ticker_range("005930", date(2026, 1, 1), date(2026, 1, 31))
    assert results == []


def test_save_preserves_created_at_on_upsert():
    repo = InvestorFlowRepository()
    first = _save(repo, "005930", date(2026, 5, 1))
    updated = _save(repo, "005930", date(2026, 5, 1), foreign_net_qty=999)
    assert updated.created_at == first.created_at
    assert updated.updated_at >= first.updated_at


def test_save_round_trips_negative_net_values():
    repo = InvestorFlowRepository()
    _save(
        repo,
        "005930",
        date(2026, 5, 2),
        foreign_net_qty=-500,
        foreign_net_krw=-2_500_000,
    )
    result = repo.get_by_ticker_and_date("005930", date(2026, 5, 2))
    assert result is not None
    assert result.foreign_net_qty == -500
    assert result.foreign_net_krw == -2_500_000
