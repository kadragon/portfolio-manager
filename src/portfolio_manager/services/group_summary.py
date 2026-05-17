"""Group summary calculation — aggregate portfolio holdings by group."""

from dataclasses import dataclass
from decimal import Decimal

from portfolio_manager.models import Group
from portfolio_manager.services.portfolio_service import PortfolioSummary


@dataclass(frozen=True)
class GroupSummaryRow:
    group: Group
    total: Decimal
    actual_pct: Decimal
    target_pct: Decimal
    diff_pct: Decimal
    diff_val: Decimal


def compute_group_summary(
    summary: PortfolioSummary,
    all_groups: list[Group] | None = None,
) -> list[GroupSummaryRow]:
    """Aggregate holdings by group using total_stock_value as denominator.

    If all_groups is provided, groups with no holdings are included (total=0).
    Rows sorted by total descending.
    """
    totals: dict[str, Decimal] = {}
    group_by_id: dict[str, Group] = {}

    for group, h in summary.holdings:
        if h.value_krw is None:
            continue
        key = str(group.id)
        totals[key] = totals.get(key, Decimal("0")) + h.value_krw
        group_by_id[key] = group

    if all_groups is not None:
        for group in all_groups:
            key = str(group.id)
            if key not in group_by_id:
                totals[key] = Decimal("0")
                group_by_id[key] = group

    denominator = summary.total_stock_value
    rows = []
    for key, total in sorted(totals.items(), key=lambda x: x[1], reverse=True):
        group = group_by_id[key]
        actual_pct = (
            total / denominator * Decimal("100") if denominator > 0 else Decimal("0")
        )
        target_pct = Decimal(str(group.target_percentage))
        diff_pct = actual_pct - target_pct
        diff_val = total - (denominator * target_pct / Decimal("100"))
        rows.append(
            GroupSummaryRow(
                group=group,
                total=total,
                actual_pct=actual_pct,
                target_pct=target_pct,
                diff_pct=diff_pct,
                diff_val=diff_val,
            )
        )
    return rows
