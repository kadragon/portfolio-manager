def test_dashboard_uses_text_badges_and_refresh_controls(client):
    response = client.get("/")

    assert response.status_code == 200
    body = response.text

    assert "대시보드" in body
    assert "badge-sell" in body
    assert "매도" in body
    assert "자동 새로고침: 켜짐" in body
    assert 'id="auto-refresh-toggle"' in body
    assert 'id="auto-refresh-poller"' in body
    assert 'data-enabled="true"' in body
    assert 'hx-target="#main-content"' in body
    assert 'hx-select="#main-content"' in body
    assert 'hx-swap="innerHTML"' in body
    assert 'hx-target="body"' not in body
    assert "📊" not in body
    assert "🔴" not in body
    assert "🟢" not in body
