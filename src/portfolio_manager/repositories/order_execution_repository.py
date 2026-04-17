"""Order execution repository."""

import json
from typing import Optional
from uuid import uuid4

from portfolio_manager.core.time import now_kst
from portfolio_manager.models.order_execution import OrderExecutionRecord
from portfolio_manager.services.database import OrderExecutionModel


class OrderExecutionRepository:
    """Repository for managing order execution records."""

    def create(
        self,
        ticker: str,
        side: str,
        quantity: int,
        currency: str,
        status: str,
        message: str,
        exchange: Optional[str] = None,
        raw_response: Optional[dict] = None,
    ) -> OrderExecutionRecord:
        """Create an order execution record."""
        now = now_kst()
        row = OrderExecutionModel.create(
            id=uuid4(),
            ticker=ticker,
            side=side,
            quantity=quantity,
            currency=currency,
            exchange=exchange,
            status=status,
            message=message,
            raw_response=json.dumps(raw_response) if raw_response is not None else None,
            created_at=now,
        )
        return self._to_domain(row)

    def list_recent(self, limit: int = 20) -> list[OrderExecutionRecord]:
        """List recent order executions ordered by created_at descending."""
        return [
            self._to_domain(row)
            for row in OrderExecutionModel.select()
            .order_by(OrderExecutionModel.created_at.desc())
            .limit(limit)
        ]

    @staticmethod
    def _to_domain(row: OrderExecutionModel) -> OrderExecutionRecord:
        raw = row.raw_response
        if isinstance(raw, str):
            raw = json.loads(raw)

        return OrderExecutionRecord(
            id=row.id,
            ticker=row.ticker,
            side=row.side,
            quantity=row.quantity,
            currency=row.currency,
            exchange=row.exchange,
            status=row.status,
            message=row.message,
            raw_response=raw,
            created_at=row.created_at,
        )
