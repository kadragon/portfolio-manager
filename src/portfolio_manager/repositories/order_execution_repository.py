"""Order execution repository."""

import json
from datetime import datetime
from typing import Optional
from uuid import UUID

from portfolio_manager.models.order_execution import OrderExecutionRecord


class OrderExecutionRepository:
    """Repository for managing order execution records."""

    def __init__(self, supabase_client):
        """Initialize the repository."""
        self.client = supabase_client

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
        data = {
            "ticker": ticker,
            "side": side,
            "quantity": quantity,
            "currency": currency,
            "exchange": exchange,
            "status": status,
            "message": message,
            "raw_response": json.dumps(raw_response)
            if raw_response is not None
            else None,
        }

        response = self.client.table("order_executions").insert(data).execute()
        item = response.data[0]

        return self._to_domain(item)

    def list_recent(self, limit: int = 20) -> list[OrderExecutionRecord]:
        """List recent order executions ordered by created_at descending."""
        response = (
            self.client.table("order_executions")
            .select("*")
            .order("created_at", desc=True)
            .limit(limit)
            .execute()
        )

        return [self._to_domain(item) for item in response.data]

    def _to_domain(self, item: dict) -> OrderExecutionRecord:
        raw = item.get("raw_response")
        if isinstance(raw, str):
            raw = json.loads(raw)

        return OrderExecutionRecord(
            id=UUID(item["id"]),
            ticker=item["ticker"],
            side=item["side"],
            quantity=item["quantity"],
            currency=item["currency"],
            exchange=item.get("exchange"),
            status=item["status"],
            message=item["message"],
            raw_response=raw,
            created_at=datetime.fromisoformat(item["created_at"]),
        )
