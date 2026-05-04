"""Cached KIS domestic investor flow for a given date."""

from dataclasses import dataclass
from datetime import date, datetime
from uuid import UUID


@dataclass
class InvestorFlow:
    id: UUID
    ticker: str
    flow_date: date
    foreign_net_qty: int
    institution_net_qty: int
    individual_net_qty: int
    foreign_net_krw: int
    institution_net_krw: int
    individual_net_krw: int
    created_at: datetime
    updated_at: datetime
