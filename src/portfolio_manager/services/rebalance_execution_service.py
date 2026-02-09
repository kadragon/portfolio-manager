"""Rebalance execution service — converts recommendations to order intents."""

from __future__ import annotations

import math
from dataclasses import dataclass
from typing import Any

from portfolio_manager.models.rebalance import RebalanceRecommendation
from portfolio_manager.services.kis.kis_market_detector import is_domestic_ticker

_DEFAULT_OVERSEAS_EXCHANGE = "NASD"


@dataclass
class OrderIntent:
    """Standardized order request before sending to KIS API."""

    ticker: str
    side: str  # "buy" or "sell"
    quantity: int
    currency: str
    exchange: str | None = None  # overseas exchange code (NASD/NYSE/AMEX)


@dataclass
class OrderIntentResult:
    """Result of creating order intents: executable intents + skipped items."""

    intents: list[OrderIntent]
    skipped: list[OrderIntent]


@dataclass
class OrderExecutionResult:
    """Result of executing a single order."""

    intent: OrderIntent
    status: str  # "success", "failed", "skipped"
    message: str = ""
    raw_response: dict | None = None


@dataclass
class RebalanceExecutionResult:
    """Full result of executing rebalance orders."""

    intents: list[OrderIntent]
    skipped: list[OrderIntent]
    executions: list[OrderExecutionResult] | None = None
    sync_warning: str | None = None


class RebalanceExecutionService:
    """Converts rebalance recommendations into executable order intents."""

    def __init__(
        self,
        order_client: Any | None = None,
        execution_repository: Any | None = None,
        sync_service: Any | None = None,
    ):
        self._order_client = order_client
        self._execution_repo = execution_repository
        self._sync_service = sync_service

    def execute_rebalance_orders(
        self,
        recommendations: list[RebalanceRecommendation],
        *,
        dry_run: bool = True,
        exchange_map: dict[str, str | None] | None = None,
    ) -> RebalanceExecutionResult:
        intent_result = self.create_order_intents(
            recommendations, exchange_map=exchange_map
        )
        executions: list[OrderExecutionResult] = []
        if not dry_run and self._order_client:
            for intent in intent_result.intents:
                resp = self._order_client.place_order(
                    ticker=intent.ticker,
                    side=intent.side,
                    quantity=intent.quantity,
                    exchange=intent.exchange,
                )
                executions.append(
                    OrderExecutionResult(
                        intent=intent,
                        status="success" if resp.get("rt_cd") == "0" else "failed",
                        message=resp.get("msg1", ""),
                        raw_response=resp,
                    )
                )
        if not dry_run and self._execution_repo:
            for ex in executions:
                self._execution_repo.create(
                    ticker=ex.intent.ticker,
                    side=ex.intent.side,
                    quantity=ex.intent.quantity,
                    currency=ex.intent.currency,
                    exchange=ex.intent.exchange,
                    status=ex.status,
                    message=ex.message,
                    raw_response=ex.raw_response,
                )
            for sk in intent_result.skipped:
                self._execution_repo.create(
                    ticker=sk.ticker,
                    side=sk.side,
                    quantity=sk.quantity,
                    currency=sk.currency,
                    exchange=sk.exchange,
                    status="skipped",
                    message="",
                    raw_response=None,
                )
        sync_warning: str | None = None
        if not dry_run and self._sync_service:
            try:
                self._sync_service.sync_account()
            except Exception as e:
                sync_warning = str(e)
        return RebalanceExecutionResult(
            intents=intent_result.intents,
            skipped=intent_result.skipped,
            executions=executions,
            sync_warning=sync_warning,
        )

    def create_order_intents(
        self,
        recommendations: list[RebalanceRecommendation],
        *,
        exchange_map: dict[str, str | None] | None = None,
    ) -> OrderIntentResult:
        intents: list[OrderIntent] = []
        skipped: list[OrderIntent] = []
        ex_map = exchange_map or {}
        for rec in recommendations:
            qty = int(math.floor(rec.quantity)) if rec.quantity is not None else 0
            exchange: str | None = None
            if not is_domestic_ticker(rec.ticker):
                exchange = ex_map.get(rec.ticker) or _DEFAULT_OVERSEAS_EXCHANGE
            intent = OrderIntent(
                ticker=rec.ticker,
                side=rec.action.value,
                quantity=qty,
                currency=rec.currency
                if rec.currency
                else "KRW"
                if is_domestic_ticker(rec.ticker)
                else "USD",
                exchange=exchange,
            )
            if qty == 0:
                skipped.append(intent)
            else:
                intents.append(intent)
        intents.sort(key=lambda i: i.side != "sell")
        return OrderIntentResult(intents=intents, skipped=skipped)
