"""AI insights routes — narrative, rebalance XAI, and Q&A partials."""

from __future__ import annotations

from fastapi import APIRouter, Form, Request
from fastapi.responses import HTMLResponse

from portfolio_manager.web.deps import get_container, get_templates

router = APIRouter(prefix="/insights")

_ALLOWED_PERIODS = {"daily", "weekly"}


def _render_unavailable(
    templates, request: Request, template_name: str
) -> HTMLResponse:
    return templates.TemplateResponse(
        request=request,
        name=template_name,
        context={
            "unavailable": True,
            "message": "AI 인사이트 서비스가 설정되지 않았습니다. OLLAMA_MODEL 환경 변수와 가격 서비스를 확인하세요.",
        },
    )


@router.get("", response_class=HTMLResponse)
def view_insights(request: Request) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    service = container.get_portfolio_insight_service()

    if service is None:
        return templates.TemplateResponse(
            request=request,
            name="insights/view.html",
            context={
                "active_page": "insights",
                "unavailable": True,
                "message": "AI 인사이트 서비스가 설정되지 않았습니다. OLLAMA_MODEL 환경 변수와 가격 서비스를 확인하세요.",
                "narrative": None,
            },
        )

    try:
        narrative = service.generate_narrative(period="daily")
        error = narrative.error
    except Exception as exc:  # noqa: BLE001
        return templates.TemplateResponse(
            request=request,
            name="insights/view.html",
            context={
                "active_page": "insights",
                "unavailable": False,
                "message": None,
                "error": str(exc),
                "narrative": None,
            },
        )

    return templates.TemplateResponse(
        request=request,
        name="insights/view.html",
        context={
            "active_page": "insights",
            "unavailable": False,
            "message": None,
            "error": error,
            "narrative": narrative,
            "current_period": "daily",
        },
    )


@router.get("/narrative", response_class=HTMLResponse)
def narrative_partial(
    request: Request,
    period: str = "daily",
) -> HTMLResponse:
    if period not in _ALLOWED_PERIODS:
        period = "daily"

    container = get_container(request)
    templates = get_templates(request)
    service = container.get_portfolio_insight_service()

    if service is None:
        return _render_unavailable(templates, request, "insights/_narrative.html")

    try:
        narrative = service.generate_narrative(period=period)  # type: ignore[arg-type]
    except Exception as exc:  # noqa: BLE001
        return templates.TemplateResponse(
            request=request,
            name="insights/_narrative.html",
            context={"error": str(exc), "narrative": None},
        )

    return templates.TemplateResponse(
        request=request,
        name="insights/_narrative.html",
        context={
            "narrative": narrative,
            "error": narrative.error,
            "current_period": period,
        },
    )


@router.get("/rebalance-xai", response_class=HTMLResponse)
def rebalance_xai_partial(request: Request) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    service = container.get_portfolio_insight_service()

    if service is None:
        return _render_unavailable(templates, request, "insights/_rebalance_xai.html")

    try:
        explanation = service.explain_rebalance()
    except Exception as exc:  # noqa: BLE001
        return templates.TemplateResponse(
            request=request,
            name="insights/_rebalance_xai.html",
            context={"error": str(exc), "explanation": None},
        )

    return templates.TemplateResponse(
        request=request,
        name="insights/_rebalance_xai.html",
        context={
            "explanation": explanation,
            "error": explanation.error,
        },
    )


@router.post("/qa", response_class=HTMLResponse)
def qa_partial(
    request: Request,
    question: str = Form(default=""),
) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    service = container.get_portfolio_insight_service()

    if service is None:
        return _render_unavailable(templates, request, "insights/_qa.html")

    question = question.strip()
    if not question:
        return templates.TemplateResponse(
            request=request,
            name="insights/_qa.html",
            context={"error": "질문을 입력하세요.", "result": None},
        )

    try:
        result = service.answer_question(question)
    except Exception as exc:  # noqa: BLE001
        return templates.TemplateResponse(
            request=request,
            name="insights/_qa.html",
            context={"error": str(exc), "result": None},
        )

    return templates.TemplateResponse(
        request=request,
        name="insights/_qa.html",
        context={"result": result, "error": result.error},
    )
