from decimal import Decimal

from portfolio_manager.models.rebalance import RebalanceAction, RebalanceRecommendation
from portfolio_manager.services.portfolio_service import PortfolioSummary
from portfolio_manager.services.rebalance_service import RebalancePlan, RegionDiagnostic
from portfolio_manager.web.routes import rebalance as rebalance_routes


def test_rebalance_page_shows_text_badges_and_execute_button(client):
    response = client.get("/rebalance")

    assert response.status_code == 200
    body = response.text

    assert '<h1 class="page-header">리밸런싱 추천</h1>' in body
    assert "진단 요약" in body
    assert "지역 비중 진단과 트리거 상태" in body
    assert "그룹별 목표 비중 대비 현재 상태" in body
    assert "badge-sell" in body
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
