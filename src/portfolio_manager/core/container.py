"""Service container for dependency injection."""

from __future__ import annotations

import os
from decimal import Decimal
from pathlib import Path

import httpx
from rich.console import Console

from portfolio_manager.repositories.account_repository import AccountRepository
from portfolio_manager.repositories.group_repository import GroupRepository
from portfolio_manager.repositories.holding_repository import HoldingRepository
from portfolio_manager.repositories.stock_repository import StockRepository
from portfolio_manager.services.kis.kis_auth_client import KisAuthClient
from portfolio_manager.services.kis.kis_domestic_info_client import (
    KisDomesticInfoClient,
)
from portfolio_manager.services.kis.kis_domestic_price_client import (
    KisDomesticPriceClient,
)
from portfolio_manager.services.kis.kis_overseas_price_client import (
    KisOverseasPriceClient,
)
from portfolio_manager.services.kis.kis_token_manager import TokenManager
from portfolio_manager.services.kis.kis_token_store import FileTokenStore
from portfolio_manager.services.kis.kis_unified_price_client import (
    KisUnifiedPriceClient,
)
from portfolio_manager.services.portfolio_service import PortfolioService
from portfolio_manager.services.price_service import PriceService
from portfolio_manager.services.exchange.exim_exchange_rate_client import (
    EximExchangeRateClient,
)
from portfolio_manager.services.exchange.exchange_rate_service import (
    ExchangeRateService,
)
from portfolio_manager.services.supabase_client import get_supabase_client


class ServiceContainer:
    """Container for services and repositories."""

    def __init__(self, console: Console | None = None) -> None:
        """Initialize container."""
        self.console = console or Console()
        self.http_client: httpx.Client | None = None
        self.exim_client: httpx.Client | None = None

        # Supabase Repositories
        self.supabase_client = get_supabase_client()
        self.group_repository = GroupRepository(self.supabase_client)
        self.stock_repository = StockRepository(self.supabase_client)
        self.account_repository = AccountRepository(self.supabase_client)
        self.holding_repository = HoldingRepository(self.supabase_client)

        # Services (initialized on demand or setup)
        self.price_service: PriceService | None = None
        self.exchange_rate_service: ExchangeRateService | None = None

    def setup(self) -> None:
        """Setup external services (KIS, Exchange)."""
        self._setup_kis_client()
        self._setup_exchange_service()

    def _setup_kis_client(self) -> None:
        app_key = os.getenv("KIS_APP_KEY")
        app_secret = os.getenv("KIS_APP_SECRET")
        env = os.getenv("KIS_ENV", "real").strip().lower()
        if "/" in env:
            env = env.split("/", 1)[0]
        cust_type = os.getenv("KIS_CUST_TYPE", "P")

        base_url = "https://openapi.koreainvestment.com:9443"
        if env in {"demo", "vps", "paper"}:
            base_url = "https://openapivts.koreainvestment.com:29443"

        self.http_client = httpx.Client(base_url=base_url)

        if app_key and app_secret:
            try:
                auth = KisAuthClient(
                    client=self.http_client, app_key=app_key, app_secret=app_secret
                )
                store = FileTokenStore(Path(".data/kis_token.json"))
                manager = TokenManager(store=store, auth_client=auth)
                token = manager.get_token()

                domestic_client = KisDomesticPriceClient(
                    client=self.http_client,
                    app_key=app_key,
                    app_secret=app_secret,
                    access_token=token,
                    cust_type=cust_type,
                    env=env,
                )
                prdt_type_cd = os.getenv("KIS_PRDT_TYPE_CD", "300").strip()
                info_tr_id = os.getenv("KIS_DOMESTIC_INFO_TR_ID", "CTPF1002R").strip()
                domestic_info_client = KisDomesticInfoClient(
                    client=self.http_client,
                    app_key=app_key,
                    app_secret=app_secret,
                    access_token=token,
                    tr_id=info_tr_id,
                    cust_type=cust_type,
                )
                overseas_client = KisOverseasPriceClient(
                    client=self.http_client,
                    app_key=app_key,
                    app_secret=app_secret,
                    access_token=token,
                    cust_type=cust_type,
                    env=env,
                )
                unified_client = KisUnifiedPriceClient(
                    domestic_client,
                    overseas_client,
                    domestic_info_client,
                    prdt_type_cd=prdt_type_cd,
                )
                self.price_service = PriceService(unified_client)
            except Exception as e:
                self.console.print(
                    f"[yellow]Warning: Could not initialize price service: {e}[/yellow]"
                )

    def _setup_exchange_service(self) -> None:
        usd_krw_rate_env = os.getenv("USD_KRW_RATE")
        exim_auth_key = os.getenv("EXIM_AUTH_KEY")

        if usd_krw_rate_env:
            self.exchange_rate_service = ExchangeRateService(
                fixed_usd_krw_rate=Decimal(usd_krw_rate_env)
            )
        elif exim_auth_key:
            self.exim_client = httpx.Client(base_url="https://oapi.koreaexim.go.kr")
            exim_rate_client = EximExchangeRateClient(
                client=self.exim_client, auth_key=exim_auth_key
            )
            self.exchange_rate_service = ExchangeRateService(
                exim_client=exim_rate_client
            )

    def get_portfolio_service(self) -> PortfolioService:
        """Create and return portfolio service."""
        return PortfolioService(
            self.group_repository,
            self.stock_repository,
            self.holding_repository,
            self.price_service,
            self.exchange_rate_service,
        )

    def close(self) -> None:
        """Close resources."""
        if self.http_client:
            self.http_client.close()
        if self.exim_client:
            self.exim_client.close()
