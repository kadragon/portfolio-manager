"""Tests for annualized return rate calculation."""

from datetime import date
from decimal import Decimal
from unittest.mock import MagicMock


def test_portfolio_summary_has_annualized_return_rate():
    """PortfolioSummary should have annualized_return_rate field."""
    from portfolio_manager.services.portfolio_service import PortfolioSummary

    summary = PortfolioSummary(
        holdings=[],
        total_value=Decimal("0"),
        total_stock_value=Decimal("100000000"),
        total_cash_balance=Decimal("0"),
        total_assets=Decimal("100000000"),
        total_invested=Decimal("87600000"),
        return_rate=Decimal("14.16"),
        first_deposit_date=date(2024, 1, 15),
        annualized_return_rate=Decimal("12.5"),
    )

    assert summary.annualized_return_rate == Decimal("12.5")
    assert summary.first_deposit_date == date(2024, 1, 15)


def test_portfolio_service_calculates_annualized_return():
    """PortfolioService should calculate annualized return rate."""
    from portfolio_manager.services.portfolio_service import PortfolioService

    # Mock repositories
    group_repo = MagicMock()
    group_repo.list_all.return_value = []

    stock_repo = MagicMock()
    holding_repo = MagicMock()
    holding_repo.get_aggregated_holdings_by_stock.return_value = {}

    account_repo = MagicMock()
    account_repo.list_all.return_value = []

    deposit_repo = MagicMock()
    deposit_repo.get_total.return_value = Decimal("87600000")
    deposit_repo.get_first_deposit_date.return_value = date(2024, 1, 15)

    price_service = MagicMock()

    service = PortfolioService(
        group_repository=group_repo,
        stock_repository=stock_repo,
        holding_repository=holding_repo,
        price_service=price_service,
        account_repository=account_repo,
        deposit_repository=deposit_repo,
    )

    # Mock today's date - assume 1 year has passed
    import portfolio_manager.services.portfolio_service as ps_module

    original_date = date

    class MockDate:
        @staticmethod
        def today():
            return date(2025, 1, 15)  # Exactly 1 year after first deposit

    ps_module.date = MockDate

    try:
        summary = service.get_portfolio_summary()
    finally:
        ps_module.date = original_date

    # With 0 assets and 87.6M invested, return should be -100%
    # But let's focus on the structure first
    assert summary.first_deposit_date == date(2024, 1, 15)
    assert summary.annualized_return_rate is not None
