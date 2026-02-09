"""Service container for dependency injection."""

from __future__ import annotations

import os
from decimal import Decimal
from pathlib import Path

import httpx
from rich.console import Console

from portfolio_manager.repositories.account_repository import AccountRepository
from portfolio_manager.repositories.deposit_repository import DepositRepository
from portfolio_manager.repositories.group_repository import GroupRepository
from portfolio_manager.repositories.holding_repository import HoldingRepository
from portfolio_manager.repositories.stock_repository import StockRepository
from portfolio_manager.repositories.order_execution_repository import (
    OrderExecutionRepository,
)
from portfolio_manager.repositories.stock_price_repository import StockPriceRepository
from portfolio_manager.services.kis.kis_auth_client import KisAuthClient
from portfolio_manager.services.kis.kis_domestic_balance_client import (
    KisDomesticBalanceClient,
)
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
from portfolio_manager.services.kis_account_sync_service import KisAccountSyncService
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
        self.stock_price_repository = StockPriceRepository(self.supabase_client)
        self.account_repository = AccountRepository(self.supabase_client)
        self.holding_repository = HoldingRepository(self.supabase_client)
        self.deposit_repository = DepositRepository(self.supabase_client)
        self.execution_repository = OrderExecutionRepository(self.supabase_client)

        # Services (initialized on demand or setup)
        self.price_service: PriceService | None = None
        self.exchange_rate_service: ExchangeRateService | None = None
        self.kis_account_sync_service: KisAccountSyncService | None = None
        self.order_client: object | None = None
        self.kis_cano: str | None = None
        self.kis_acnt_prdt_cd: str | None = None

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
                    token_manager=manager,
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
                    token_manager=manager,
                )
                unified_client = KisUnifiedPriceClient(
                    domestic_client,
                    overseas_client,
                    domestic_info_client,
                    prdt_type_cd=prdt_type_cd,
                )
                self.price_service = PriceService(
                    unified_client,
                    price_cache_repository=self.stock_price_repository,
                )

                cano, acnt_prdt_cd = self._load_kis_account_credentials()
                if cano and acnt_prdt_cd:
                    balance_client = KisDomesticBalanceClient(
                        client=self.http_client,
                        app_key=app_key,
                        app_secret=app_secret,
                        access_token=token,
                        cust_type=cust_type,
                        env=env,
                        token_manager=manager,
                    )
                    self.kis_account_sync_service = KisAccountSyncService(
                        account_repository=self.account_repository,
                        holding_repository=self.holding_repository,
                        stock_repository=self.stock_repository,
                        group_repository=self.group_repository,
                        kis_balance_client=balance_client,
                    )
                    self.kis_cano = cano
                    self.kis_acnt_prdt_cd = acnt_prdt_cd
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
            account_repository=self.account_repository,
            deposit_repository=self.deposit_repository,
        )

    def close(self) -> None:
        """Close resources."""
        if self.http_client:
            self.http_client.close()
        if self.exim_client:
            self.exim_client.close()

    @staticmethod
    def _load_kis_account_credentials() -> tuple[str | None, str | None]:
        cano = os.getenv("KIS_CANO", "").strip()
        acnt_prdt_cd = os.getenv("KIS_ACNT_PRDT_CD", "").strip()
        if cano and acnt_prdt_cd:
            return cano, acnt_prdt_cd

        account_no = os.getenv("KIS_ACCOUNT_NO", "").strip()
        if account_no:
            digits = "".join(ch for ch in account_no if ch.isdigit())
            if len(digits) == 10:
                return digits[:8], digits[8:]

        return None, None
