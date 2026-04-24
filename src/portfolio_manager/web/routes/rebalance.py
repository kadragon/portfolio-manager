"""Rebalance routes — recommendations + order execution."""

from fastapi import APIRouter, Form, Request
from fastapi.responses import HTMLResponse

from portfolio_manager.services.rebalance_execution_service import (
    RebalanceExecutionService,
)
from portfolio_manager.services.portfolio_service import PortfolioSummary
from portfolio_manager.services.rebalance_service import RebalancePlan, RebalanceService
from portfolio_manager.web.deps import get_container, get_templates

router = APIRouter(prefix="/rebalance")


def _build_rebalance_plan(container) -> tuple[PortfolioSummary, RebalancePlan]:
    portfolio_service = container.get_portfolio_service()
    summary = portfolio_service.get_portfolio_summary(include_change_rates=False)

    accounts = container.account_repository.list_all()
    holdings_by_account = {
        account.id: container.holding_repository.list_by_account(account.id)
        for account in accounts
    }
    groups = container.group_repository.list_all()
    stocks = container.stock_repository.list_all()

    rebalance_service = RebalanceService()
    plan = rebalance_service.build_plan(
        summary=summary,
        accounts=accounts,
        holdings_by_account=holdings_by_account,
        groups=groups,
        stocks=stocks,
    )
    return summary, plan


@router.get("", response_class=HTMLResponse)
def view_rebalance(request: Request) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)

    if not container.price_service:
        return templates.TemplateResponse(
            request=request,
            name="rebalance/view.html",
            context={
                "active_page": "rebalance",
                "error": "가격 서비스가 설정되지 않았습니다. KIS API 키를 확인하세요.",
                "sell_recommendations": [],
                "buy_recommendations": [],
                "account_summaries": [],
                "summary": None,
                "plan": None,
            },
        )

    try:
        summary, plan = _build_rebalance_plan(container)
        error = None
    except Exception as e:
        return templates.TemplateResponse(
            request=request,
            name="rebalance/view.html",
            context={
                "active_page": "rebalance",
                "error": str(e),
                "sell_recommendations": [],
                "buy_recommendations": [],
                "account_summaries": [],
                "summary": None,
                "plan": None,
            },
        )

    return templates.TemplateResponse(
        request=request,
        name="rebalance/view.html",
        context={
            "active_page": "rebalance",
            "error": error,
            "sell_recommendations": plan.sell_recommendations,
            "buy_recommendations": plan.buy_recommendations,
            "account_summaries": plan.account_summaries,
            "summary": summary,
            "plan": plan,
            "has_order_client": container.order_client is not None,
        },
    )


@router.post("/execute", response_class=HTMLResponse)
def execute_rebalance(
    request: Request,
    confirm: str = Form(default=""),
) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)

    if not container.price_service:
        return templates.TemplateResponse(
            request=request,
            name="rebalance/_result.html",
            context={"success": False, "message": "가격 서비스 없음", "result": None},
        )

    try:
        _summary, plan = _build_rebalance_plan(container)
        all_recs = plan.sell_recommendations + plan.buy_recommendations

        # Build exchange map from stocks
        stocks = container.stock_repository.list_all()
        exchange_map: dict[str, str | None] = {
            s.ticker: s.exchange for s in stocks if s.exchange
        }

        execution_service = RebalanceExecutionService(
            order_client=container.order_client,
            execution_repository=container.execution_repository,
            sync_service=container.kis_account_sync_service,
        )
        result = execution_service.execute_rebalance_orders(
            all_recs, dry_run=False, exchange_map=exchange_map
        )
        return templates.TemplateResponse(
            request=request,
            name="rebalance/_result.html",
            context={"success": True, "message": "주문 실행 완료", "result": result},
        )
    except Exception as e:
        return templates.TemplateResponse(
            request=request,
            name="rebalance/_result.html",
            context={"success": False, "message": str(e), "result": None},
        )
