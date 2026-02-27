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
