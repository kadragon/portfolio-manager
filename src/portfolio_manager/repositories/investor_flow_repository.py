"""Investor flow repository for database operations."""

from __future__ import annotations

from datetime import date
from uuid import uuid4

from peewee import IntegrityError

from portfolio_manager.core.time import now_kst
from portfolio_manager.models.investor_flow import InvestorFlow
from portfolio_manager.services.database import InvestorFlowModel


class InvestorFlowRepository:
    """Repository for cached KIS domestic investor flow data."""

    def get_by_ticker_and_date(
        self, ticker: str, flow_date: date
    ) -> InvestorFlow | None:
        row = InvestorFlowModel.get_or_none(
            (InvestorFlowModel.ticker == ticker)
            & (InvestorFlowModel.flow_date == flow_date)
        )
        return self._to_domain(row) if row else None

    def save(
        self,
        *,
        ticker: str,
        flow_date: date,
        foreign_net_qty: int,
        institution_net_qty: int,
        individual_net_qty: int,
        foreign_net_krw: int,
        institution_net_krw: int,
        individual_net_krw: int,
    ) -> InvestorFlow:
        now = now_kst()
        fields = dict(
            ticker=ticker,
            flow_date=flow_date,
            foreign_net_qty=foreign_net_qty,
            institution_net_qty=institution_net_qty,
            individual_net_qty=individual_net_qty,
            foreign_net_krw=foreign_net_krw,
            institution_net_krw=institution_net_krw,
            individual_net_krw=individual_net_krw,
        )
        try:
            row = InvestorFlowModel.create(
                id=uuid4(), created_at=now, updated_at=now, **fields
            )
        except IntegrityError:
            InvestorFlowModel.update(**fields, updated_at=now).where(
                (InvestorFlowModel.ticker == ticker)
                & (InvestorFlowModel.flow_date == flow_date)
            ).execute()
            row = InvestorFlowModel.get(
                (InvestorFlowModel.ticker == ticker)
                & (InvestorFlowModel.flow_date == flow_date)
            )
        return self._to_domain(row)

    def list_by_ticker_range(
        self, ticker: str, start: date, end: date
    ) -> list[InvestorFlow]:
        rows = (
            InvestorFlowModel.select()
            .where(
                (InvestorFlowModel.ticker == ticker)
                & (InvestorFlowModel.flow_date >= start)
                & (InvestorFlowModel.flow_date <= end)
            )
            .order_by(InvestorFlowModel.flow_date.asc())
        )
        return [self._to_domain(row) for row in rows]

    @staticmethod
    def _to_domain(row: InvestorFlowModel) -> InvestorFlow:
        return InvestorFlow(
            id=row.id,
            ticker=row.ticker,
            flow_date=row.flow_date,
            foreign_net_qty=row.foreign_net_qty,
            institution_net_qty=row.institution_net_qty,
            individual_net_qty=row.individual_net_qty,
            foreign_net_krw=row.foreign_net_krw,
            institution_net_krw=row.institution_net_krw,
            individual_net_krw=row.individual_net_krw,
            created_at=row.created_at,
            updated_at=row.updated_at,
        )
