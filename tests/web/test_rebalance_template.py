from decimal import Decimal


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


def test_rebalance_page_shows_fractional_quantity_for_overseas(client, fake_container):
    fake_container.group.name = "해외성장"
    fake_container.group.target_percentage = 35.0
    fake_container.stock.ticker = "AAPL"
    fake_container.holding.quantity = Decimal("0.5")
    summary_holding = fake_container.portfolio_service.summary.holdings[0][1]
    summary_holding.stock.ticker = "AAPL"
    summary_holding.quantity = Decimal("0.5")
    summary_holding.price = Decimal("100")
    summary_holding.currency = "USD"
    summary_holding.value_krw = Decimal("700000")

    response = client.get("/rebalance")

    assert response.status_code == 200
    body = response.text
    assert "0.214286" in body
