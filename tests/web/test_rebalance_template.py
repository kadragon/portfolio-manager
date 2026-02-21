def test_rebalance_page_shows_text_badges_and_execute_button(client):
    response = client.get("/rebalance")

    assert response.status_code == 200
    body = response.text

    assert "리밸런싱 추천" in body
    assert "badge-sell" in body
    assert "주문 실행" in body
    assert "📊" not in body
    assert "🔴" not in body
    assert "🟢" not in body


def test_rebalance_execute_partial_has_live_region(client):
    response = client.post("/rebalance/execute")

    assert response.status_code == 200
    body = response.text

    assert 'id="execute-result"' in body
    assert 'aria-live="polite"' in body
