"""Accounts + Holdings CRUD routes."""

from decimal import Decimal
from uuid import UUID

from fastapi import APIRouter, Form, Request
from fastapi.responses import HTMLResponse, Response

from portfolio_manager.web.deps import get_container, get_templates

router = APIRouter(prefix="/accounts")


@router.get("", response_class=HTMLResponse)
def list_accounts(request: Request) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    accounts = container.account_repository.list_all()
    return templates.TemplateResponse(
        request=request,
        name="accounts/list.html",
        context={"accounts": accounts, "active_page": "accounts"},
    )


@router.get("/{account_id}", response_class=HTMLResponse)
def get_account_row(request: Request, account_id: UUID) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    accounts = container.account_repository.list_all()
    account = next((a for a in accounts if a.id == account_id), None)
    if account is None:
        return Response(status_code=404)  # type: ignore[return-value]
    return templates.TemplateResponse(
        request=request,
        name="accounts/_row.html",
        context={"account": account},
    )


@router.post("", response_class=HTMLResponse)
def create_account(
    request: Request,
    name: str = Form(...),
    cash_balance: str = Form("0"),
) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    account = container.account_repository.create(
        name=name.strip(),
        cash_balance=Decimal(cash_balance or "0"),
    )
    return templates.TemplateResponse(
        request=request,
        name="accounts/_row.html",
        context={"account": account},
    )


@router.get("/{account_id}/edit", response_class=HTMLResponse)
def edit_account_form(request: Request, account_id: UUID) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    accounts = container.account_repository.list_all()
    account = next((a for a in accounts if a.id == account_id), None)
    if account is None:
        return Response(status_code=404)  # type: ignore[return-value]
    return templates.TemplateResponse(
        request=request,
        name="accounts/_form.html",
        context={"account": account},
    )


@router.put("/{account_id}", response_class=HTMLResponse)
def update_account(
    request: Request,
    account_id: UUID,
    name: str = Form(...),
    cash_balance: str = Form("0"),
) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    account = container.account_repository.update(
        account_id=account_id,
        name=name.strip(),
        cash_balance=Decimal(cash_balance or "0"),
    )
    return templates.TemplateResponse(
        request=request,
        name="accounts/_row.html",
        context={"account": account},
    )


@router.delete("/{account_id}")
def delete_account(request: Request, account_id: UUID) -> Response:
    container = get_container(request)
    container.account_repository.delete_with_holdings(
        account_id, container.holding_repository
    )
    return Response(status_code=200)


# ── Holdings ──────────────────────────────────────────────────────────────────


@router.get("/{account_id}/holdings", response_class=HTMLResponse)
def list_holdings(request: Request, account_id: UUID) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    accounts = container.account_repository.list_all()
    account = next((a for a in accounts if a.id == account_id), None)
    if account is None:
        return Response(status_code=404)  # type: ignore[return-value]
    holdings = container.holding_repository.list_by_account(account_id)
    stocks = container.stock_repository.list_all()
    stock_map = {s.id: s for s in stocks}
    all_stocks = stocks
    return templates.TemplateResponse(
        request=request,
        name="accounts/holdings.html",
        context={
            "account": account,
            "holdings": holdings,
            "stock_map": stock_map,
            "all_stocks": all_stocks,
            "active_page": "accounts",
        },
    )


@router.post("/{account_id}/holdings", response_class=HTMLResponse)
def create_holding(
    request: Request,
    account_id: UUID,
    stock_id: str = Form(...),
    quantity: str = Form(...),
) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    holding = container.holding_repository.create(
        account_id=account_id,
        stock_id=UUID(stock_id),
        quantity=Decimal(quantity),
    )
    stocks = container.stock_repository.list_all()
    stock_map = {s.id: s for s in stocks}
    return templates.TemplateResponse(
        request=request,
        name="accounts/_holding_row.html",
        context={"holding": holding, "stock_map": stock_map, "account_id": account_id},
    )


@router.get("/{account_id}/holdings/{holding_id}", response_class=HTMLResponse)
def get_holding_row(
    request: Request, account_id: UUID, holding_id: UUID
) -> HTMLResponse:
    """Return a single holding row partial (used by cancel in edit form)."""
    container = get_container(request)
    templates = get_templates(request)
    holdings = container.holding_repository.list_by_account(account_id)
    holding = next((h for h in holdings if h.id == holding_id), None)
    if holding is None:
        return Response(status_code=404)  # type: ignore[return-value]
    stocks = container.stock_repository.list_all()
    stock_map = {s.id: s for s in stocks}
    return templates.TemplateResponse(
        request=request,
        name="accounts/_holding_row.html",
        context={"holding": holding, "stock_map": stock_map, "account_id": account_id},
    )


@router.get("/{account_id}/holdings/{holding_id}/edit", response_class=HTMLResponse)
def edit_holding_form(
    request: Request, account_id: UUID, holding_id: UUID
) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    holdings = container.holding_repository.list_by_account(account_id)
    holding = next((h for h in holdings if h.id == holding_id), None)
    if holding is None:
        return Response(status_code=404)  # type: ignore[return-value]
    stocks = container.stock_repository.list_all()
    stock_map = {s.id: s for s in stocks}
    return templates.TemplateResponse(
        request=request,
        name="accounts/_holding_form.html",
        context={"holding": holding, "stock_map": stock_map, "account_id": account_id},
    )


@router.put("/{account_id}/holdings/{holding_id}", response_class=HTMLResponse)
def update_holding(
    request: Request,
    account_id: UUID,
    holding_id: UUID,
    quantity: str = Form(...),
) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    holding = container.holding_repository.update(
        holding_id=holding_id, quantity=Decimal(quantity)
    )
    stocks = container.stock_repository.list_all()
    stock_map = {s.id: s for s in stocks}
    return templates.TemplateResponse(
        request=request,
        name="accounts/_holding_row.html",
        context={"holding": holding, "stock_map": stock_map, "account_id": account_id},
    )


@router.delete("/{account_id}/holdings/{holding_id}")
def delete_holding(request: Request, account_id: UUID, holding_id: UUID) -> Response:
    container = get_container(request)
    container.holding_repository.delete(holding_id)
    return Response(status_code=200)


# ── KIS Sync ─────────────────────────────────────────────────────────────────


@router.post("/{account_id}/sync", response_class=HTMLResponse)
def sync_account(request: Request, account_id: UUID) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)

    if container.kis_account_sync_service is None:
        return templates.TemplateResponse(
            request=request,
            name="accounts/_sync_result.html",
            context={
                "account_id": account_id,
                "success": False,
                "message": "KIS 계좌 동기화 서비스가 설정되지 않았습니다. (.env에 KIS_CANO/KIS_ACNT_PRDT_CD 확인)",
            },
        )

    try:
        accounts = container.account_repository.list_all()
        account = next((a for a in accounts if a.id == account_id), None)
        if account is None:
            return templates.TemplateResponse(
                request=request,
                name="accounts/_sync_result.html",
                context={
                    "account_id": account_id,
                    "success": False,
                    "message": "계좌를 찾을 수 없습니다.",
                },
            )
        cano = container.kis_cano or ""
        acnt = container.kis_acnt_prdt_cd or ""
        container.kis_account_sync_service.sync_account(
            account=account, cano=cano, acnt_prdt_cd=acnt
        )
        message = "KIS 계좌 동기화 완료"
        success = True
    except Exception as e:
        message = f"동기화 실패: {e}"
        success = False

    return templates.TemplateResponse(
        request=request,
        name="accounts/_sync_result.html",
        context={"account_id": account_id, "success": success, "message": message},
    )
