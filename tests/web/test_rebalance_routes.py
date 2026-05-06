class _RebalanceSummaryFailService:
    def get_portfolio_summary(
        self, *, include_change_rates=True, change_rate_periods=None
    ):
        raise RuntimeError("rebalance summary failed")


def test_view_rebalance_shows_error_when_price_service_is_missing(
    client, fake_container
):
    fake_container.price_service = None

    response = client.get("/rebalance")

    assert response.status_code == 200
    assert "가격 서비스가 설정되지 않았습니다." in response.text


def test_view_rebalance_shows_error_when_summary_fetch_fails(client, fake_container):
    fake_container.portfolio_service = _RebalanceSummaryFailService()

    response = client.get("/rebalance")

    assert response.status_code == 200
    assert "rebalance summary failed" in response.text


def test_execute_rebalance_returns_failure_when_price_service_is_missing(
    client, fake_container
):
    fake_container.price_service = None

    response = client.post("/rebalance/execute")

    assert response.status_code == 200
    assert "가격 서비스 없음" in response.text


def test_execute_rebalance_returns_failure_when_summary_fetch_raises(
    client, fake_container
):
    fake_container.portfolio_service = _RebalanceSummaryFailService()

    response = client.post("/rebalance/execute")

    assert response.status_code == 200
    assert "rebalance summary failed" in response.text


def test_view_rebalance_shows_error_when_group_mapping_is_invalid(
    client, fake_container
):
    fake_container.group.name = "국내 주식"

    response = client.get("/rebalance")

    assert response.status_code == 200
    assert "리밸런싱 그룹 매핑 불가 그룹" in response.text


def test_view_rebalance_restrict_overseas_renders_checkbox_checked_and_banner(
    client, fake_container
):
    response = client.get("/rebalance?restrict_overseas=on")

    assert response.status_code == 200
    assert 'name="restrict_overseas"' in response.text
    assert "checked" in response.text
    assert "해외 매도 제한 옵션 적용 중" in response.text


def test_execute_rebalance_passes_restrict_overseas_flag(
    client, fake_container, monkeypatch
):
    from portfolio_manager.web.routes import rebalance as rebalance_routes

    captured: dict = {}
    original = rebalance_routes._build_rebalance_plan

    def _spy(container, **kwargs):
        captured.update(kwargs)
        return original(container, **kwargs)

    monkeypatch.setattr(rebalance_routes, "_build_rebalance_plan", _spy)

    response = client.post("/rebalance/execute", data={"restrict_overseas": "on"})

    assert response.status_code == 200
    assert captured.get("restrict_overseas") is True
