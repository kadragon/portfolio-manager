"""Deposits CRUD routes."""

from datetime import date
from decimal import Decimal
from uuid import UUID

from fastapi import APIRouter, Form, Request
from fastapi.responses import HTMLResponse, Response

from portfolio_manager.web.deps import get_container, get_templates

router = APIRouter(prefix="/deposits")


@router.get("", response_class=HTMLResponse)
def list_deposits(request: Request) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    deposits = container.deposit_repository.list_all()
    total = container.deposit_repository.get_total()
    return templates.TemplateResponse(
        request=request,
        name="deposits/list.html",
        context={"deposits": deposits, "total": total, "active_page": "deposits"},
    )


@router.get("/{deposit_id}", response_class=HTMLResponse)
def get_deposit_row(request: Request, deposit_id: UUID) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    deposits = container.deposit_repository.list_all()
    deposit = next((d for d in deposits if d.id == deposit_id), None)
    if deposit is None:
        return Response(status_code=404)  # type: ignore[return-value]
    return templates.TemplateResponse(
        request=request,
        name="deposits/_row.html",
        context={"deposit": deposit},
    )


@router.post("", response_class=HTMLResponse)
def create_deposit(
    request: Request,
    amount: str = Form(...),
    deposit_date: str = Form(...),
    note: str = Form(""),
) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)

    parsed_date = date.fromisoformat(deposit_date)

    # Check for duplicate date
    existing = container.deposit_repository.get_by_date(parsed_date)
    if existing is not None:
        # Update instead of creating
        deposit = container.deposit_repository.update(
            deposit_id=existing.id,
            amount=Decimal(amount),
            deposit_date=parsed_date,
            note=note.strip() or None,
        )
        return templates.TemplateResponse(
            request=request,
            name="deposits/_row.html",
            context={"deposit": deposit},
            headers={
                "HX-Reswap": "none"
            },  # don't insert a new row; client should reload
        )

    deposit = container.deposit_repository.create(
        amount=Decimal(amount),
        deposit_date=parsed_date,
        note=note.strip() or None,
    )
    return templates.TemplateResponse(
        request=request,
        name="deposits/_row.html",
        context={"deposit": deposit},
    )


@router.get("/{deposit_id}/edit", response_class=HTMLResponse)
def edit_deposit_form(request: Request, deposit_id: UUID) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    deposits = container.deposit_repository.list_all()
    deposit = next((d for d in deposits if d.id == deposit_id), None)
    if deposit is None:
        return Response(status_code=404)  # type: ignore[return-value]
    return templates.TemplateResponse(
        request=request,
        name="deposits/_form.html",
        context={"deposit": deposit},
    )


@router.put("/{deposit_id}", response_class=HTMLResponse)
def update_deposit(
    request: Request,
    deposit_id: UUID,
    amount: str = Form(...),
    deposit_date: str = Form(...),
    note: str = Form(""),
) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    deposit = container.deposit_repository.update(
        deposit_id=deposit_id,
        amount=Decimal(amount),
        deposit_date=date.fromisoformat(deposit_date),
        note=note.strip() or None,
    )
    return templates.TemplateResponse(
        request=request,
        name="deposits/_row.html",
        context={"deposit": deposit},
    )


@router.delete("/{deposit_id}")
def delete_deposit(request: Request, deposit_id: UUID) -> Response:
    container = get_container(request)
    container.deposit_repository.delete(deposit_id)
    return Response(status_code=200)
