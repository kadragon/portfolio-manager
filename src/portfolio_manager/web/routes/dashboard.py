"""Dashboard route — GET /"""

from fastapi import APIRouter, Request
from fastapi.responses import HTMLResponse

from portfolio_manager.services.group_summary import compute_group_summary
from portfolio_manager.web.deps import get_container, get_templates

router = APIRouter()


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
                include_change_rates=True,
                change_rate_periods=("1y", "6m", "1m"),
            )
            group_summary = compute_group_summary(summary)
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
