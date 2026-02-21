"""Dashboard route — GET /"""

from collections import defaultdict
from decimal import Decimal

from fastapi import APIRouter, Request
from fastapi.responses import HTMLResponse

from portfolio_manager.models import Group
from portfolio_manager.services.portfolio_service import PortfolioSummary
from portfolio_manager.web.deps import get_container, get_templates

router = APIRouter()


def _compute_group_summary(summary: PortfolioSummary) -> list[dict]:
    """Compute per-group totals for the group summary table."""
    group_totals: dict[str, Decimal] = defaultdict(Decimal)
    group_lookup: dict[str, Group] = {}

    for group, h in summary.holdings:
        key = str(group.id)
        value = h.value_krw if h.value_krw is not None else h.value
        group_totals[key] += value
        group_lookup[key] = group

    total_value = summary.total_value
    rows = []
    for key, total in sorted(group_totals.items(), key=lambda x: x[1], reverse=True):
        group = group_lookup[key]
        actual_pct = (total / total_value * 100) if total_value > 0 else Decimal("0")
        target_pct = Decimal(str(group.target_percentage))
        diff_pct = actual_pct - target_pct
        diff_val = total - (total_value * target_pct / Decimal("100"))
        rows.append(
            {
                "group": group,
                "total": total,
                "actual_pct": actual_pct,
                "target_pct": target_pct,
                "diff_pct": diff_pct,
                "diff_val": diff_val,
            }
        )
    return rows


@router.get("/", response_class=HTMLResponse)
def dashboard(request: Request) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    portfolio_service = container.get_portfolio_service()

    summary = None
    group_holdings = None
    group_summary = None
    error = None

    if container.price_service:
        try:
            summary = portfolio_service.get_portfolio_summary(
                include_change_rates=False
            )
            group_summary = _compute_group_summary(summary)
        except Exception as e:
            error = str(e)
            try:
                group_holdings = portfolio_service.get_holdings_by_group()
            except Exception:
                group_holdings = None
    else:
        try:
            group_holdings = portfolio_service.get_holdings_by_group()
        except Exception as e:
            error = str(e)

    return templates.TemplateResponse(
        request=request,
        name="dashboard.html",
        context={
            "summary": summary,
            "group_holdings": group_holdings,
            "group_summary": group_summary,
            "error": error,
            "active_page": "dashboard",
        },
    )
