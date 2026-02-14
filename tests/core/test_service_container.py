"""Tests for ServiceContainer wiring and setup branches."""

from __future__ import annotations

from decimal import Decimal
import os
from typing import Iterator, cast
from unittest.mock import MagicMock, patch

import pytest

from portfolio_manager.core.container import ServiceContainer


@pytest.fixture
def container() -> Iterator[ServiceContainer]:
    """Create a container with repositories mocked out."""
    with (
        patch(
            "portfolio_manager.core.container.get_supabase_client",
            return_value=MagicMock(),
        ),
        patch(
            "portfolio_manager.core.container.GroupRepository", return_value=MagicMock()
        ),
        patch(
            "portfolio_manager.core.container.StockRepository", return_value=MagicMock()
        ),
        patch(
            "portfolio_manager.core.container.StockPriceRepository",
            return_value=MagicMock(),
        ),
        patch(
            "portfolio_manager.core.container.AccountRepository",
            return_value=MagicMock(),
        ),
        patch(
            "portfolio_manager.core.container.HoldingRepository",
            return_value=MagicMock(),
        ),
        patch(
            "portfolio_manager.core.container.DepositRepository",
            return_value=MagicMock(),
        ),
        patch(
            "portfolio_manager.core.container.OrderExecutionRepository",
            return_value=MagicMock(),
        ),
    ):
        yield ServiceContainer(console=MagicMock())


def test_constructor_wires_repositories_and_defaults() -> None:
    """Container should wire repository instances and initialize defaults."""
    console = MagicMock()
    supabase_client = MagicMock()
    group_repo = MagicMock()
    stock_repo = MagicMock()
    stock_price_repo = MagicMock()
    account_repo = MagicMock()
    holding_repo = MagicMock()
    deposit_repo = MagicMock()
    execution_repo = MagicMock()

    with (
        patch(
            "portfolio_manager.core.container.get_supabase_client",
            return_value=supabase_client,
        ),
        patch(
            "portfolio_manager.core.container.GroupRepository", return_value=group_repo
        ),
        patch(
            "portfolio_manager.core.container.StockRepository", return_value=stock_repo
        ),
        patch(
            "portfolio_manager.core.container.StockPriceRepository",
            return_value=stock_price_repo,
        ),
        patch(
            "portfolio_manager.core.container.AccountRepository",
            return_value=account_repo,
        ),
        patch(
            "portfolio_manager.core.container.HoldingRepository",
            return_value=holding_repo,
        ),
        patch(
            "portfolio_manager.core.container.DepositRepository",
            return_value=deposit_repo,
        ),
        patch(
            "portfolio_manager.core.container.OrderExecutionRepository",
            return_value=execution_repo,
        ),
    ):
        created = ServiceContainer(console=console)

    assert created.console is console
    assert created.supabase_client is supabase_client
    assert created.group_repository is group_repo
    assert created.stock_repository is stock_repo
    assert created.stock_price_repository is stock_price_repo
    assert created.account_repository is account_repo
    assert created.holding_repository is holding_repo
    assert created.deposit_repository is deposit_repo
    assert created.execution_repository is execution_repo
    assert created.price_service is None
    assert created.exchange_rate_service is None
    assert created.kis_account_sync_service is None
    assert created.order_client is None
    assert created.kis_cano is None
    assert created.kis_acnt_prdt_cd is None


def test_setup_calls_kis_and_exchange_setup(container: ServiceContainer) -> None:
    """setup should run both KIS and exchange setup methods."""
    with (
        patch.object(container, "_setup_kis_client") as setup_kis,
        patch.object(container, "_setup_exchange_service") as setup_exchange,
    ):
        container.setup()

    setup_kis.assert_called_once_with()
    setup_exchange.assert_called_once_with()


def test_setup_kis_client_without_credentials_keeps_price_service_none(
    container: ServiceContainer,
) -> None:
    """Without KIS credentials, KIS clients should not be initialized."""
    with (
        patch.dict(
            os.environ,
            {"KIS_APP_KEY": "", "KIS_APP_SECRET": "", "KIS_ENV": "real"},
            clear=False,
        ),
        patch(
            "portfolio_manager.core.container.httpx.Client", return_value=MagicMock()
        ),
    ):
        container._setup_kis_client()

    assert container.price_service is None


def test_setup_kis_client_normalizes_env_real_prod(container: ServiceContainer) -> None:
    """real/prod should normalize to real API base URL."""
    with (
        patch.dict(
            os.environ,
            {"KIS_APP_KEY": "", "KIS_APP_SECRET": "", "KIS_ENV": "real/prod"},
            clear=False,
        ),
        patch(
            "portfolio_manager.core.container.httpx.Client", return_value=MagicMock()
        ) as client_cls,
    ):
        container._setup_kis_client()

    assert (
        client_cls.call_args.kwargs["base_url"]
        == "https://openapi.koreainvestment.com:9443"
    )


def test_setup_kis_client_normalizes_env_demo_paper(
    container: ServiceContainer,
) -> None:
    """demo/vps/paper should normalize to mock trading API base URL."""
    with (
        patch.dict(
            os.environ,
            {"KIS_APP_KEY": "", "KIS_APP_SECRET": "", "KIS_ENV": "demo/vps/paper"},
            clear=False,
        ),
        patch(
            "portfolio_manager.core.container.httpx.Client", return_value=MagicMock()
        ) as client_cls,
    ):
        container._setup_kis_client()

    assert (
        client_cls.call_args.kwargs["base_url"]
        == "https://openapivts.koreainvestment.com:29443"
    )


def test_setup_kis_client_initializes_price_service(
    container: ServiceContainer,
) -> None:
    """With credentials, price service should be initialized."""
    token_manager = MagicMock()
    token_manager.get_token.return_value = "token"
    unified_client = MagicMock()
    price_service = MagicMock()

    with (
        patch.dict(
            os.environ,
            {
                "KIS_APP_KEY": "app-key",
                "KIS_APP_SECRET": "app-secret",
                "KIS_ENV": "real",
            },
            clear=False,
        ),
        patch(
            "portfolio_manager.core.container.httpx.Client", return_value=MagicMock()
        ),
        patch("portfolio_manager.core.container.KisAuthClient"),
        patch("portfolio_manager.core.container.FileTokenStore"),
        patch(
            "portfolio_manager.core.container.TokenManager", return_value=token_manager
        ),
        patch(
            "portfolio_manager.core.container.KisDomesticPriceClient",
            return_value=MagicMock(),
        ),
        patch(
            "portfolio_manager.core.container.KisDomesticInfoClient",
            return_value=MagicMock(),
        ),
        patch(
            "portfolio_manager.core.container.KisOverseasPriceClient",
            return_value=MagicMock(),
        ),
        patch(
            "portfolio_manager.core.container.KisUnifiedPriceClient",
            return_value=unified_client,
        ),
        patch(
            "portfolio_manager.core.container.PriceService",
            return_value=price_service,
        ) as price_service_cls,
        patch.object(
            ServiceContainer,
            "_load_kis_account_credentials",
            return_value=(None, None),
        ),
    ):
        container._setup_kis_client()

    assert container.price_service is price_service
    price_service_cls.assert_called_once_with(
        unified_client,
        price_cache_repository=container.stock_price_repository,
    )


def test_setup_kis_client_wires_sync_and_order_clients_when_account_exists(
    container: ServiceContainer,
) -> None:
    """Account credentials should wire sync service and order client."""
    token_manager = MagicMock()
    token_manager.get_token.return_value = "token"
    price_service = MagicMock()
    order_client = MagicMock()

    with (
        patch.dict(
            os.environ,
            {
                "KIS_APP_KEY": "app-key",
                "KIS_APP_SECRET": "app-secret",
                "KIS_ENV": "real",
            },
            clear=False,
        ),
        patch(
            "portfolio_manager.core.container.httpx.Client", return_value=MagicMock()
        ),
        patch("portfolio_manager.core.container.KisAuthClient"),
        patch("portfolio_manager.core.container.FileTokenStore"),
        patch(
            "portfolio_manager.core.container.TokenManager", return_value=token_manager
        ),
        patch(
            "portfolio_manager.core.container.KisDomesticPriceClient",
            return_value=MagicMock(),
        ),
        patch(
            "portfolio_manager.core.container.KisDomesticInfoClient",
            return_value=MagicMock(),
        ),
        patch(
            "portfolio_manager.core.container.KisOverseasPriceClient",
            return_value=MagicMock(),
        ),
        patch(
            "portfolio_manager.core.container.KisUnifiedPriceClient",
            return_value=MagicMock(),
        ),
        patch(
            "portfolio_manager.core.container.PriceService", return_value=price_service
        ),
        patch(
            "portfolio_manager.core.container.KisDomesticBalanceClient",
            return_value=MagicMock(),
        ),
        patch(
            "portfolio_manager.core.container.KisAccountSyncService",
            return_value=MagicMock(),
        ),
        patch(
            "portfolio_manager.core.container.KisDomesticOrderClient",
            return_value=MagicMock(),
        ),
        patch(
            "portfolio_manager.core.container.KisOverseasOrderClient",
            return_value=MagicMock(),
        ),
        patch(
            "portfolio_manager.core.container.KisUnifiedOrderClient",
            return_value=order_client,
        ),
        patch.object(
            ServiceContainer,
            "_load_kis_account_credentials",
            return_value=("12345678", "01"),
        ),
    ):
        container._setup_kis_client()

    assert container.kis_account_sync_service is not None
    assert container.kis_cano == "12345678"
    assert container.kis_acnt_prdt_cd == "01"
    assert container.order_client is order_client


def test_setup_kis_client_prints_warning_on_exception(
    container: ServiceContainer,
) -> None:
    """Client setup failures should be handled and logged as warning."""
    token_manager = MagicMock()
    token_manager.get_token.side_effect = RuntimeError("boom")

    with (
        patch.dict(
            os.environ,
            {
                "KIS_APP_KEY": "app-key",
                "KIS_APP_SECRET": "app-secret",
                "KIS_ENV": "real",
            },
            clear=False,
        ),
        patch(
            "portfolio_manager.core.container.httpx.Client", return_value=MagicMock()
        ),
        patch("portfolio_manager.core.container.KisAuthClient"),
        patch("portfolio_manager.core.container.FileTokenStore"),
        patch(
            "portfolio_manager.core.container.TokenManager", return_value=token_manager
        ),
    ):
        container._setup_kis_client()

    console_print = cast(MagicMock, container.console.print)
    console_print.assert_called()
    assert "Could not initialize price service" in str(console_print.call_args)


def test_setup_exchange_service_uses_fixed_rate(container: ServiceContainer) -> None:
    """USD_KRW_RATE should initialize fixed exchange service."""
    exchange_service = MagicMock()
    with (
        patch.dict(
            os.environ, {"USD_KRW_RATE": "1350.25", "EXIM_AUTH_KEY": ""}, clear=False
        ),
        patch(
            "portfolio_manager.core.container.ExchangeRateService",
            return_value=exchange_service,
        ) as exchange_service_cls,
    ):
        container._setup_exchange_service()

    assert container.exchange_rate_service is exchange_service
    exchange_service_cls.assert_called_once_with(fixed_usd_krw_rate=Decimal("1350.25"))


def test_setup_exchange_service_uses_exim_when_key_present(
    container: ServiceContainer,
) -> None:
    """EXIM auth key should initialize EXIM-backed exchange service."""
    exim_http_client = MagicMock()
    exim_rate_client = MagicMock()
    exchange_service = MagicMock()
    with (
        patch.dict(
            os.environ, {"USD_KRW_RATE": "", "EXIM_AUTH_KEY": "auth-key"}, clear=False
        ),
        patch(
            "portfolio_manager.core.container.httpx.Client",
            return_value=exim_http_client,
        ) as http_client_cls,
        patch(
            "portfolio_manager.core.container.EximExchangeRateClient",
            return_value=exim_rate_client,
        ) as exim_client_cls,
        patch(
            "portfolio_manager.core.container.ExchangeRateService",
            return_value=exchange_service,
        ) as exchange_service_cls,
    ):
        container._setup_exchange_service()

    assert container.exim_client is exim_http_client
    assert container.exchange_rate_service is exchange_service
    http_client_cls.assert_called_once_with(base_url="https://oapi.koreaexim.go.kr")
    exim_client_cls.assert_called_once_with(
        client=exim_http_client, auth_key="auth-key"
    )
    exchange_service_cls.assert_called_once_with(exim_client=exim_rate_client)


def test_get_portfolio_service_passes_dependencies(
    container: ServiceContainer,
) -> None:
    """Portfolio service should be created with expected dependencies."""
    container.price_service = MagicMock()
    container.exchange_rate_service = MagicMock()
    portfolio_service = MagicMock()

    with patch(
        "portfolio_manager.core.container.PortfolioService",
        return_value=portfolio_service,
    ) as service_cls:
        created = container.get_portfolio_service()

    assert created is portfolio_service
    service_cls.assert_called_once_with(
        container.group_repository,
        container.stock_repository,
        container.holding_repository,
        container.price_service,
        container.exchange_rate_service,
        account_repository=container.account_repository,
        deposit_repository=container.deposit_repository,
    )


def test_close_closes_http_clients(container: ServiceContainer) -> None:
    """close should close both managed HTTP clients when present."""
    container.http_client = MagicMock()
    container.exim_client = MagicMock()

    container.close()

    container.http_client.close.assert_called_once_with()
    container.exim_client.close.assert_called_once_with()


def test_load_kis_account_credentials_prefers_cano_and_product_code() -> None:
    """Direct KIS_CANO and KIS_ACNT_PRDT_CD should be returned."""
    with patch.dict(
        os.environ,
        {"KIS_CANO": "12345678", "KIS_ACNT_PRDT_CD": "01", "KIS_ACCOUNT_NO": ""},
        clear=False,
    ):
        assert ServiceContainer._load_kis_account_credentials() == ("12345678", "01")


def test_load_kis_account_credentials_parses_account_no_digits() -> None:
    """KIS_ACCOUNT_NO should be parsed when direct values are absent."""
    with patch.dict(
        os.environ,
        {"KIS_CANO": "", "KIS_ACNT_PRDT_CD": "", "KIS_ACCOUNT_NO": "1234-5678-01"},
        clear=False,
    ):
        assert ServiceContainer._load_kis_account_credentials() == ("12345678", "01")


def test_load_kis_account_credentials_returns_none_when_missing() -> None:
    """Missing credentials should return (None, None)."""
    with patch.dict(
        os.environ,
        {"KIS_CANO": "", "KIS_ACNT_PRDT_CD": "", "KIS_ACCOUNT_NO": "invalid"},
        clear=False,
    ):
        assert ServiceContainer._load_kis_account_credentials() == (None, None)
