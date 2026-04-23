from decimal import Decimal

from portfolio_manager.models.rebalance import (
    AccountRebalanceSummary,
    RebalanceAction,
    RebalanceRecommendation,
)
from portfolio_manager.services.portfolio_service import PortfolioSummary
from portfolio_manager.services.rebalance_service import RebalancePlan, RegionDiagnostic
from portfolio_manager.web.routes import rebalance as rebalance_routes


def _make_region_diagnostic() -> RegionDiagnostic:
    return RegionDiagnostic(
        target_kr_percentage=Decimal("50"),
        target_us_percentage=Decimal("50"),
        current_kr_percentage=Decimal("50"),
        current_us_percentage=Decimal("50"),
        lower_kr_percentage=Decimal("45"),
        upper_kr_percentage=Decimal("55"),
        is_triggered=False,
    )


def test_rebalance_page_shows_text_badges_and_execute_button(client):
    response = client.get("/rebalance")

    assert response.status_code == 200
    body = response.text

    assert ">리밸런싱 추천</h1>" in body
    assert "진단 요약" in body
    assert "지역 비중 진단과 트리거 상태" in body
    assert "그룹별 목표 비중 대비 현재 상태" in body
    assert "badge-error" in body
    assert "주문 실행" in body
    assert 'hx-disabled-elt="this"' in body
    assert "📊" not in body
    assert "🔴" not in body
    assert "🟢" not in body


def test_rebalance_execute_partial_has_live_region(client):
    response = client.post("/rebalance/execute")

    assert response.status_code == 200
    body = response.text

    assert 'id="execute-result"' in body
    assert 'role="status"' in body
    assert 'aria-live="polite"' in body


def test_rebalance_page_shows_fractional_quantity_for_overseas(client, monkeypatch):
    recommendation = RebalanceRecommendation(
        ticker="AAPL",
        action=RebalanceAction.BUY,
        amount=Decimal("100"),
        priority=1,
        currency="USD",
        quantity=Decimal("0.214286"),
        stock_name="Apple Inc",
        group_name="해외성장",
        account_name="메인 계좌",
        rebalance_group_name="해외성장",
        reason="테스트 추천",
        trigger_type="group",
        amount_krw=Decimal("130000"),
    )

    plan = RebalancePlan(
        sell_recommendations=[],
        buy_recommendations=[recommendation],
        group_diagnostics=[],
        region_diagnostic=RegionDiagnostic(
            target_kr_percentage=Decimal("50"),
            target_us_percentage=Decimal("50"),
            current_kr_percentage=Decimal("50"),
            current_us_percentage=Decimal("50"),
            lower_kr_percentage=Decimal("45"),
            upper_kr_percentage=Decimal("55"),
            is_triggered=False,
        ),
        total_assets_krw=Decimal("1000000"),
    )

    summary = PortfolioSummary(holdings=[], total_value=Decimal("0"))

    monkeypatch.setattr(
        rebalance_routes,
        "_build_rebalance_plan",
        lambda _container: (summary, plan),
    )

    response = client.get("/rebalance")

    assert response.status_code == 200
    body = response.text
    assert "0.214286" in body


def test_rebalance_page_renders_account_sections(client, monkeypatch):
    rec_sell = RebalanceRecommendation(
        ticker="005930",
        action=RebalanceAction.SELL,
        amount=Decimal("100000"),
        priority=1,
        currency="KRW",
        quantity=Decimal("10"),
        stock_name="삼성전자",
        group_name="국내성장",
        account_name="ISA계좌",
        rebalance_group_name="국내성장",
        amount_krw=Decimal("100000"),
    )
    rec_buy = RebalanceRecommendation(
        ticker="QQQ",
        action=RebalanceAction.BUY,
        amount=Decimal("50"),
        priority=1,
        currency="USD",
        quantity=Decimal("0.1"),
        stock_name="Invesco QQQ",
        group_name="해외성장",
        account_name="해외계좌",
        rebalance_group_name="해외성장",
        amount_krw=Decimal("70000"),
    )

    plan = RebalancePlan(
        sell_recommendations=[rec_sell],
        buy_recommendations=[rec_buy],
        group_diagnostics=[],
        region_diagnostic=_make_region_diagnostic(),
        total_assets_krw=Decimal("1000000"),
        account_summaries=[
            AccountRebalanceSummary(
                account_id="aaa",
                account_name="ISA계좌",
                starting_cash_krw=Decimal("0"),
                sell_cash_krw=Decimal("100000"),
                total_sell_krw=Decimal("100000"),
                total_buy_krw=Decimal("0"),
                unused_cash_krw=Decimal("100000"),
                unmet_groups=[],
                sell_recommendations=[rec_sell],
                buy_recommendations=[],
            ),
            AccountRebalanceSummary(
                account_id="bbb",
                account_name="해외계좌",
                starting_cash_krw=Decimal("70000"),
                sell_cash_krw=Decimal("0"),
                total_sell_krw=Decimal("0"),
                total_buy_krw=Decimal("70000"),
                unused_cash_krw=Decimal("0"),
                unmet_groups=[],
                sell_recommendations=[],
                buy_recommendations=[rec_buy],
            ),
        ],
    )

    monkeypatch.setattr(
        rebalance_routes,
        "_build_rebalance_plan",
        lambda _container: (
            PortfolioSummary(holdings=[], total_value=Decimal("0")),
            plan,
        ),
    )

    response = client.get("/rebalance")
    assert response.status_code == 200
    body = response.text

    assert "ISA계좌" in body
    assert "해외계좌" in body
    assert "삼성전자" in body
    assert "Invesco QQQ" in body


def test_rebalance_page_shows_unmet_groups_warning(client, monkeypatch):
    plan = RebalancePlan(
        sell_recommendations=[],
        buy_recommendations=[],
        group_diagnostics=[],
        region_diagnostic=_make_region_diagnostic(),
        total_assets_krw=Decimal("500000"),
        account_summaries=[
            AccountRebalanceSummary(
                account_id="ccc",
                account_name="국내계좌",
                starting_cash_krw=Decimal("50000"),
                sell_cash_krw=Decimal("0"),
                total_sell_krw=Decimal("0"),
                total_buy_krw=Decimal("0"),
                unused_cash_krw=Decimal("50000"),
                unmet_groups=["해외성장", "해외안정", "해외배당"],
                sell_recommendations=[],
                buy_recommendations=[],
            ),
        ],
    )

    monkeypatch.setattr(
        rebalance_routes,
        "_build_rebalance_plan",
        lambda _container: (
            PortfolioSummary(holdings=[], total_value=Decimal("0")),
            plan,
        ),
    )

    response = client.get("/rebalance")
    assert response.status_code == 200
    body = response.text

    assert "다음 회차로 이월됩니다" in body
    assert "해외성장" in body
    assert "해외안정" in body
    assert "해외배당" in body
