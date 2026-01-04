"""Test portfolio service."""

from decimal import Decimal
from unittest.mock import Mock
from uuid import uuid4

from portfolio_manager.models import Group, Stock
from portfolio_manager.services.portfolio_service import PortfolioService


def test_get_holdings_by_group():
    """그룹별로 주식과 합산 수량을 조회한다."""
    # Given: 그룹과 주식
    group1_id = uuid4()
    group2_id = uuid4()
    stock1_id = uuid4()
    stock2_id = uuid4()

    group_repo = Mock()
    group_repo.list_all.return_value = [
        Group(
            id=group1_id,
            name="Tech",
            created_at=None,  # type: ignore[arg-type]
            updated_at=None,  # type: ignore[arg-type]
        ),
        Group(
            id=group2_id,
            name="Finance",
            created_at=None,  # type: ignore[arg-type]
            updated_at=None,  # type: ignore[arg-type]
        ),
    ]

    stock_repo = Mock()
    stock_repo.list_by_group.side_effect = lambda group_id: (
        [
            Stock(
                id=stock1_id,
                ticker="AAPL",
                group_id=group1_id,
                created_at=None,  # type: ignore[arg-type]
                updated_at=None,  # type: ignore[arg-type]
            )
        ]
        if group_id == group1_id
        else [
            Stock(
                id=stock2_id,
                ticker="JPM",
                group_id=group2_id,
                created_at=None,  # type: ignore[arg-type]
                updated_at=None,  # type: ignore[arg-type]
            )
        ]
    )

    holding_repo = Mock()
    holding_repo.get_aggregated_holdings_by_stock.return_value = {
        stock1_id: Decimal("15"),
        stock2_id: Decimal("20"),
    }

    service = PortfolioService(group_repo, stock_repo, holding_repo)

    # When: 그룹별 보유 현황을 조회
    result = service.get_holdings_by_group()

    # Then: 그룹별로 주식과 합산 수량이 반환됨
    assert len(result) == 2
    assert result[0].group.id == group1_id
    assert len(result[0].stock_holdings) == 1
    assert result[0].stock_holdings[0].stock.ticker == "AAPL"
    assert result[0].stock_holdings[0].quantity == Decimal("15")
    assert result[1].group.id == group2_id
    assert len(result[1].stock_holdings) == 1
    assert result[1].stock_holdings[0].stock.ticker == "JPM"
    assert result[1].stock_holdings[0].quantity == Decimal("20")
