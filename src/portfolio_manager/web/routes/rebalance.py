"""Rebalance routes — recommendations + order execution."""

from fastapi import APIRouter, Form, Query, Request
from fastapi.responses import HTMLResponse

from portfolio_manager.services.rebalance_execution_service import (
    RebalanceExecutionService,
)
from portfolio_manager.services.portfolio_service import PortfolioSummary
from portfolio_manager.services.rebalance_service import RebalancePlan, RebalanceService
from portfolio_manager.web.deps import get_container, get_templates

router = APIRouter(prefix="/rebalance")


def _build_rebalance_plan(
    container, restrict_overseas: bool = False
) -> tuple[PortfolioSummary, RebalancePlan]:
    return RebalanceService().build_plan_from_repos(
        portfolio_service=container.get_portfolio_service(),
        account_repository=container.account_repository,
        holding_repository=container.holding_repository,
        group_repository=container.group_repository,
        stock_repository=container.stock_repository,
        restrict_overseas=restrict_overseas,
    )


@router.get("", response_class=HTMLResponse)
def view_rebalance(
    request: Request,
    restrict_overseas: bool = Query(False),
) -> HTMLResponse:
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
                "restrict_overseas": restrict_overseas,
            },
        )

    try:
        summary, plan = _build_rebalance_plan(
            container, restrict_overseas=restrict_overseas
        )
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
                "restrict_overseas": restrict_overseas,
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
            "restrict_overseas": restrict_overseas,
        },
    )


@router.post("/execute", response_class=HTMLResponse)
def execute_rebalance(
    request: Request,
    confirm: str = Form(default=""),
    restrict_overseas: bool = Form(False),
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
        _summary, plan = _build_rebalance_plan(
            container, restrict_overseas=restrict_overseas
        )
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
