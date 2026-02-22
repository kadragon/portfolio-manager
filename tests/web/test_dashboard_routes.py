class _SummaryFailsPortfolioService:
    def __init__(self, group_holdings):
        self._group_holdings = group_holdings

    def get_portfolio_summary(
        self,
        *,
        include_change_rates=True,
        change_rate_periods=None,
    ):
        raise RuntimeError("summary failed")

    def get_holdings_by_group(self):
        return self._group_holdings


class _AllFailsPortfolioService:
    def get_portfolio_summary(
        self,
        *,
        include_change_rates=True,
        change_rate_periods=None,
    ):
        raise RuntimeError("summary failed")

    def get_holdings_by_group(self):
        raise RuntimeError("holdings failed")


class _HoldingsFailPortfolioService:
    def get_portfolio_summary(
        self, *, include_change_rates=True, change_rate_periods=None
    ):
        return None

    def get_holdings_by_group(self):
        raise RuntimeError("holdings failed")


def test_dashboard_falls_back_to_group_holdings_when_summary_fails(
    client, fake_container
):
    fake_container.portfolio_service = _SummaryFailsPortfolioService(
        fake_container.portfolio_service.group_holdings
    )

    response = client.get("/")

    assert response.status_code == 200
    assert "summary failed" in response.text
    assert fake_container.stock.ticker in response.text


def test_dashboard_shows_error_when_summary_and_holdings_fallback_fail(
    client, fake_container
):
    fake_container.portfolio_service = _AllFailsPortfolioService()

    response = client.get("/")

    assert response.status_code == 200
    assert "summary failed" in response.text
    assert "포트폴리오 데이터를 불러올 수 없습니다." in response.text


def test_dashboard_shows_error_when_price_service_missing_and_holdings_fail(
    client, fake_container
):
    fake_container.price_service = None
    fake_container.portfolio_service = _HoldingsFailPortfolioService()

    response = client.get("/")

    assert response.status_code == 200
    assert "holdings failed" in response.text
    assert "포트폴리오 데이터를 불러올 수 없습니다." in response.text
