"""Service container for dependency injection."""

from __future__ import annotations

import logging
import os
from dataclasses import dataclass
from decimal import Decimal
from pathlib import Path

import httpx

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
from portfolio_manager.services.kis.kis_overseas_info_client import (
    KisOverseasInfoClient,
)
from portfolio_manager.services.kis.kis_overseas_price_client import (
    KisOverseasPriceClient,
)
from portfolio_manager.services.kis.kis_token_manager import TokenManager
from portfolio_manager.services.kis.kis_token_store import FileTokenStore
from portfolio_manager.services.kis.kis_unified_price_client import (
    KisUnifiedPriceClient,
)
from portfolio_manager.services.kis.kis_domestic_order_client import (
    KisDomesticOrderClient,
)
from portfolio_manager.services.kis.kis_overseas_order_client import (
    KisOverseasOrderClient,
)
from portfolio_manager.services.kis.kis_unified_order_client import (
    KisUnifiedOrderClient,
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
from portfolio_manager.services.database import init_db, close_db
from portfolio_manager.services.llm.ollama_client import OllamaClient
from portfolio_manager.services.stock_service import StockService
from portfolio_manager.services.portfolio_insight_service import (
    PortfolioInsightService,
)

logger = logging.getLogger(__name__)


@dataclass
class KisClientSet:
    """A set of KIS API clients sharing the same app_key/app_secret."""

    balance_client: KisDomesticBalanceClient
    domestic_order_client: KisDomesticOrderClient
    overseas_order_client: KisOverseasOrderClient


class ServiceContainer:
    """Container for services and repositories."""

    def __init__(self) -> None:
        """Initialize container."""
        self.http_client: httpx.Client | None = None
        self.exim_client: httpx.Client | None = None
        self.ollama_http_client: httpx.Client | None = None

        # SQLite + Peewee
        init_db()
        self.group_repository = GroupRepository()
        self.stock_repository = StockRepository()
        self.stock_price_repository = StockPriceRepository()
        self.account_repository = AccountRepository()
        self.holding_repository = HoldingRepository()
        self.deposit_repository = DepositRepository()
        self.execution_repository = OrderExecutionRepository()

        # Services (initialized on demand or setup)
        self.price_service: PriceService | None = None
        self.stock_service: StockService = StockService(self.stock_repository)
        self.exchange_rate_service: ExchangeRateService | None = None
        self.kis_account_sync_service: KisAccountSyncService | None = None
        self.order_client: object | None = None
        self.kis_cano: str | None = None
        self.kis_acnt_prdt_cd: str | None = None
        self.kis_client_sets: dict[int, KisClientSet] = {}
        self.ollama_client: OllamaClient | None = None

    def setup(self) -> None:
        """Setup external services (KIS, Exchange, Ollama)."""
        self._setup_kis_client()
        self._setup_exchange_service()
        self._setup_ollama_client()

    def _setup_ollama_client(self) -> None:
        host = os.getenv("OLLAMA_HOST", "http://localhost:11434").strip()
        model = os.getenv("OLLAMA_MODEL", "").strip()
        if not model:
            return
        try:
            timeout_sec = float(os.getenv("OLLAMA_TIMEOUT_SEC", "60"))
        except ValueError:
            timeout_sec = 60.0
        num_ctx_raw = os.getenv("OLLAMA_NUM_CTX")
        num_ctx: int | None = None
        if num_ctx_raw:
            try:
                num_ctx = int(num_ctx_raw)
            except ValueError:
                num_ctx = None
        self.ollama_http_client = httpx.Client(base_url=host)
        self.ollama_client = OllamaClient(
            client=self.ollama_http_client,
            model=model,
            timeout_sec=timeout_sec,
            num_ctx=num_ctx,
        )

    def get_portfolio_insight_service(self) -> PortfolioInsightService | None:
        """Return a PortfolioInsightService if LLM + price service are configured."""
        if self.ollama_client is None or self.price_service is None:
            return None
        return PortfolioInsightService(
            portfolio_service=self.get_portfolio_service(),
            account_repository=self.account_repository,
            holding_repository=self.holding_repository,
            group_repository=self.group_repository,
            stock_repository=self.stock_repository,
            deposit_repository=self.deposit_repository,
            ollama_client=self.ollama_client,
        )

    def _build_kis_client_set(
        self,
        key_id: int,
        app_key: str,
        app_secret: str,
        http_client: httpx.Client,
        cust_type: str,
        env: str,
    ) -> tuple[KisClientSet, TokenManager, str]:
        """Build a KIS client set for a given API key pair.

        Returns (client_set, token_manager, access_token) so the caller
        can reuse the manager/token for price clients on key set 1.
        """
        auth = KisAuthClient(client=http_client, app_key=app_key, app_secret=app_secret)
        store = FileTokenStore(Path(f".data/kis_token_{key_id}.json"))
        manager = TokenManager(store=store, auth_client=auth)
        token = manager.get_token()

        balance_client = KisDomesticBalanceClient(
            client=http_client,
            app_key=app_key,
            app_secret=app_secret,
            access_token=token,
            cust_type=cust_type,
            env=env,
            token_manager=manager,
        )
        domestic_order_client = KisDomesticOrderClient(
            client=http_client,
            app_key=app_key,
            app_secret=app_secret,
            access_token=token,
            cust_type=cust_type,
            env=env,
            token_manager=manager,
        )
        overseas_order_client = KisOverseasOrderClient(
            client=http_client,
            app_key=app_key,
            app_secret=app_secret,
            access_token=token,
            cust_type=cust_type,
            env=env,
            token_manager=manager,
        )
        client_set = KisClientSet(
            balance_client=balance_client,
            domestic_order_client=domestic_order_client,
            overseas_order_client=overseas_order_client,
        )
        return client_set, manager, token

    def get_kis_client_set(self, key_id: int | None) -> KisClientSet | None:
        """Get a KIS client set by key ID (None defaults to 1)."""
        effective_id = key_id or 1
        return self.kis_client_sets.get(effective_id)

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
                client_set_1, manager, token = self._build_kis_client_set(
                    1, app_key, app_secret, self.http_client, cust_type, env
                )
                self.kis_client_sets[1] = client_set_1

                # Price clients use key set 1 only
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
                overseas_info_tr_id = os.getenv(
                    "KIS_OVERSEAS_INFO_TR_ID", "CTPF1702R"
                ).strip()
                overseas_info_client = KisOverseasInfoClient(
                    client=self.http_client,
                    app_key=app_key,
                    app_secret=app_secret,
                    access_token=token,
                    tr_id=overseas_info_tr_id,
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
                    overseas_info_client=overseas_info_client,
                )
                self.price_service = PriceService(
                    unified_client,
                    price_cache_repository=self.stock_price_repository,
                )
                self.stock_service.set_price_service(self.price_service)

                cano, acnt_prdt_cd = self._load_kis_account_credentials()
                if cano and acnt_prdt_cd:
                    self.kis_account_sync_service = KisAccountSyncService(
                        account_repository=self.account_repository,
                        holding_repository=self.holding_repository,
                        stock_repository=self.stock_repository,
                        group_repository=self.group_repository,
                        kis_balance_client=client_set_1.balance_client,
                        sync_log_path=Path(".data/kis_sync.log"),
                        stock_service=self.stock_service,
                    )
                    self.kis_cano = cano
                    self.kis_acnt_prdt_cd = acnt_prdt_cd

                    self.order_client = KisUnifiedOrderClient(
                        domestic_client=client_set_1.domestic_order_client,
                        overseas_client=client_set_1.overseas_order_client,
                        cano=cano,
                        acnt_prdt_cd=acnt_prdt_cd,
                        price_service=self.price_service,
                    )

                # Key set 2 (optional)
                app_key_2 = os.getenv("KIS_APP_KEY_2")
                app_secret_2 = os.getenv("KIS_APP_SECRET_2")
                if app_key_2 and app_secret_2:
                    try:
                        client_set_2, _, _ = self._build_kis_client_set(
                            2,
                            app_key_2,
                            app_secret_2,
                            self.http_client,
                            cust_type,
                            env,
                        )
                        self.kis_client_sets[2] = client_set_2
                    except Exception as e:
                        logger.warning("Could not initialize KIS key set 2: %s", e)

            except Exception as e:
                logger.warning("Could not initialize price service: %s", e)

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
            stock_service=self.stock_service,
            price_service=self.price_service,
            exchange_rate_service=self.exchange_rate_service,
            account_repository=self.account_repository,
            deposit_repository=self.deposit_repository,
        )

    def close(self) -> None:
        """Close resources."""
        close_db()
        if self.http_client:
            self.http_client.close()
        if self.exim_client:
            self.exim_client.close()
        if self.ollama_http_client:
            self.ollama_http_client.close()

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
