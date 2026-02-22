def test_dashboard_uses_text_badges_and_compact_summary_layout(client):
    response = client.get("/")

    assert response.status_code == 200
    body = response.text

    assert "대시보드" in body
    assert "투자 요약" in body
    assert "badge-sell" in body
    assert "매도" in body
    assert "자동 새로고침" not in body
    assert 'id="auto-refresh-toggle"' not in body
    assert 'id="auto-refresh-poller"' not in body
    assert ">1Y<" in body
    assert ">6M<" in body
    assert ">1M<" in body
    assert body.index("투자 요약") < body.index("보유 종목")
    assert 'hx-target="body"' not in body
    assert "📊" not in body
    assert "🔴" not in body
    assert "🟢" not in body
