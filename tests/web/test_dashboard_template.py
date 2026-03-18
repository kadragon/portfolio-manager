def test_dashboard_uses_text_badges_and_compact_summary_layout(client):
    response = client.get("/")

    assert response.status_code == 200
    body = response.text

    assert '<h1 class="page-header">대시보드</h1>' in body
    assert "summary-bar" in body
    assert "summary-item" in body
    assert "그룹별 보유 종목, 현재가, 평가액과 기간 수익률" in body
    assert "그룹별 평가액, 현재 비중, 목표 대비 차이와 권장 동작" in body
    assert 'tabindex="0"' in body
    assert "dashboard-rate-periods" in body
    assert "badge-sell" in body
    assert "매도" in body
    assert "자동 새로고침" not in body
    assert 'id="auto-refresh-toggle"' not in body
    assert 'id="auto-refresh-poller"' not in body
    assert ">1Y<" in body
    assert ">6M<" in body
    assert ">1M<" in body
    assert 'hx-target="body"' not in body
    assert "📊" not in body
    assert "🔴" not in body
    assert "🟢" not in body
