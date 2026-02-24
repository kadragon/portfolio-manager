def test_rebalance_page_shows_text_badges_and_execute_button(client):
    response = client.get("/rebalance")

    assert response.status_code == 200
    body = response.text

    assert '<h1 class="page-header">리밸런싱 추천</h1>' in body
    assert "진단 요약" in body
    assert "지역 비중 진단과 트리거 상태" in body
    assert "슬리브별 목표 비중 대비 현재 상태" in body
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
