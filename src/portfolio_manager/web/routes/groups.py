"""Groups CRUD routes."""

from uuid import UUID

from fastapi import APIRouter, Form, Request
from fastapi.responses import HTMLResponse, Response

from portfolio_manager.web.deps import get_container, get_templates

router = APIRouter(prefix="/groups")


def _is_htmx(request: Request) -> bool:
    return request.headers.get("HX-Request") == "true"


@router.get("", response_class=HTMLResponse)
def list_groups(request: Request) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    groups = container.group_repository.list_all()
    return templates.TemplateResponse(
        request=request,
        name="groups/list.html",
        context={"groups": groups, "active_page": "groups"},
    )


@router.get("/{group_id}", response_class=HTMLResponse)
def get_group_row(request: Request, group_id: UUID) -> HTMLResponse:
    """Return a single group row partial (used by cancel in edit form)."""
    container = get_container(request)
    templates = get_templates(request)
    groups = container.group_repository.list_all()
    group = next((g for g in groups if g.id == group_id), None)
    if group is None:
        return Response(status_code=404)  # type: ignore[return-value]
    return templates.TemplateResponse(
        request=request,
        name="groups/_row.html",
        context={"group": group},
    )


@router.post("", response_class=HTMLResponse)
def create_group(
    request: Request,
    name: str = Form(...),
    target_percentage: float = Form(0.0),
) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    group = container.group_repository.create(
        name=name.strip(), target_percentage=target_percentage
    )
    return templates.TemplateResponse(
        request=request,
        name="groups/_row.html",
        context={"group": group},
    )


@router.get("/{group_id}/edit", response_class=HTMLResponse)
def edit_group_form(request: Request, group_id: UUID) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    groups = container.group_repository.list_all()
    group = next((g for g in groups if g.id == group_id), None)
    if group is None:
        return Response(status_code=404)  # type: ignore[return-value]
    return templates.TemplateResponse(
        request=request,
        name="groups/_form.html",
        context={"group": group},
    )


@router.put("/{group_id}", response_class=HTMLResponse)
def update_group(
    request: Request,
    group_id: UUID,
    name: str = Form(...),
    target_percentage: float = Form(0.0),
) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    group = container.group_repository.update(
        group_id=group_id,
        name=name.strip(),
        target_percentage=target_percentage,
    )
    return templates.TemplateResponse(
        request=request,
        name="groups/_row.html",
        context={"group": group},
    )


@router.delete("/{group_id}")
def delete_group(request: Request, group_id: UUID) -> Response:
    container = get_container(request)
    container.group_repository.delete(group_id)
    return Response(status_code=200)


# ── Stocks within group ──────────────────────────────────────────────────────


@router.get("/{group_id}/stocks", response_class=HTMLResponse)
def list_stocks(request: Request, group_id: UUID) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    groups = container.group_repository.list_all()
    group = next((g for g in groups if g.id == group_id), None)
    if group is None:
        return Response(status_code=404)  # type: ignore[return-value]
    stocks = container.stock_repository.list_by_group(group_id)
    return templates.TemplateResponse(
        request=request,
        name="groups/stocks.html",
        context={"group": group, "stocks": stocks, "active_page": "groups"},
    )


@router.post("/{group_id}/stocks", response_class=HTMLResponse)
def create_stock(
    request: Request,
    group_id: UUID,
    ticker: str = Form(...),
) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    stock = container.stock_repository.create(
        ticker=ticker.strip().upper(), group_id=group_id
    )
    return templates.TemplateResponse(
        request=request,
        name="groups/_stock_row.html",
        context={"stock": stock, "group_id": group_id},
    )


@router.delete("/{group_id}/stocks/{stock_id}")
def delete_stock(request: Request, group_id: UUID, stock_id: UUID) -> Response:
    container = get_container(request)
    container.stock_repository.delete(stock_id)
    return Response(status_code=200)
