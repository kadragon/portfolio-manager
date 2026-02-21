"""Rebalance routes — recommendations + order execution."""

from fastapi import APIRouter, Form, Request
from fastapi.responses import HTMLResponse

from portfolio_manager.services.rebalance_execution_service import (
    RebalanceExecutionService,
)
from portfolio_manager.services.rebalance_service import RebalanceService
from portfolio_manager.web.deps import get_container, get_templates

router = APIRouter(prefix="/rebalance")


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
                "summary": None,
            },
        )

    try:
        portfolio_service = container.get_portfolio_service()
        summary = portfolio_service.get_portfolio_summary(include_change_rates=False)
        rebalance_service = RebalanceService()
        sell_recommendations = rebalance_service.get_sell_recommendations(summary)
        buy_recommendations = rebalance_service.get_buy_recommendations(summary)
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
                "summary": None,
            },
        )

    return templates.TemplateResponse(
        request=request,
        name="rebalance/view.html",
        context={
            "active_page": "rebalance",
            "error": error,
            "sell_recommendations": sell_recommendations,
            "buy_recommendations": buy_recommendations,
            "summary": summary,
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
        portfolio_service = container.get_portfolio_service()
        summary = portfolio_service.get_portfolio_summary(include_change_rates=False)
        rebalance_service = RebalanceService()
        sell_recs = rebalance_service.get_sell_recommendations(summary)
        buy_recs = rebalance_service.get_buy_recommendations(summary)
        all_recs = sell_recs + buy_recs

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
