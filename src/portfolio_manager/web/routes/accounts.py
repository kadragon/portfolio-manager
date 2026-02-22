"""Accounts + Holdings CRUD routes."""

from decimal import Decimal
from uuid import UUID

from fastapi import APIRouter, Form, Request
from fastapi.responses import HTMLResponse, Response
from postgrest.exceptions import APIError

from portfolio_manager.web.deps import get_container, get_templates

router = APIRouter(prefix="/accounts")


def _build_stock_map(container) -> dict:
    stocks = container.stock_repository.list_all()
    return {stock.id: stock for stock in stocks}


def _format_stock_name(name: str) -> str:
    return name.replace("증권상장지수투자신탁(주식)", "").strip()


def _build_stock_name_map(container, stocks: list | None = None) -> dict:
    stock_items = container.stock_repository.list_all() if stocks is None else stocks

    price_service = getattr(container, "price_service", None)
    if price_service is None or not hasattr(price_service, "get_stock_price"):
        return {}

    stock_name_map = {}
    for stock in stock_items:
        resolved_name = ""
        try:
            _, _, resolved_name, _ = price_service.get_stock_price(
                stock.ticker, preferred_exchange=stock.exchange
            )
        except (APIError, ValueError, TypeError):
            resolved_name = ""
        if resolved_name:
            stock_name_map[stock.id] = _format_stock_name(resolved_name)
    return stock_name_map


def _render_holdings_rows(request: Request, templates, container, account_id: UUID):
    holdings = container.holding_repository.list_by_account(account_id)
    stocks = container.stock_repository.list_all()
    stock_map = {stock.id: stock for stock in stocks}
    stock_name_map = _build_stock_name_map(container, stocks)
    return templates.TemplateResponse(
        request=request,
        name="accounts/_holdings_rows.html",
        context={
            "holdings": holdings,
            "stock_map": stock_map,
            "stock_name_map": stock_name_map,
            "account_id": account_id,
        },
    )


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
    cash_balance: Decimal = Form(Decimal("0")),
) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    account = container.account_repository.create(
        name=name.strip(),
        cash_balance=cash_balance,
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
    cash_balance: Decimal = Form(Decimal("0")),
) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    account = container.account_repository.update(
        account_id=account_id,
        name=name.strip(),
        cash_balance=cash_balance,
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
    all_stocks = container.stock_repository.list_all()
    stock_map = {s.id: s for s in all_stocks}
    stock_name_map = _build_stock_name_map(container, all_stocks)
    return templates.TemplateResponse(
        request=request,
        name="accounts/holdings.html",
        context={
            "account": account,
            "holdings": holdings,
            "stock_map": stock_map,
            "stock_name_map": stock_name_map,
            "all_stocks": all_stocks,
            "active_page": "accounts",
        },
    )


@router.post("/{account_id}/holdings", response_class=HTMLResponse)
def create_holding(
    request: Request,
    account_id: UUID,
    stock_id: UUID = Form(...),
    quantity: Decimal = Form(...),
) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    container.holding_repository.create(
        account_id=account_id,
        stock_id=stock_id,
        quantity=quantity,
    )
    return _render_holdings_rows(request, templates, container, account_id)


@router.put("/{account_id}/holdings/bulk", response_class=HTMLResponse)
def bulk_update_holdings(
    request: Request,
    account_id: UUID,
    holding_id: list[UUID] = Form(default=[]),
    quantity: list[Decimal] = Form(default=[]),
) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)

    def _error(message: str) -> HTMLResponse:
        return templates.TemplateResponse(
            request=request,
            name="accounts/_holdings_bulk_result.html",
            context={
                "success": False,
                "message": message,
                "holdings": None,
                "stock_map": None,
                "account_id": account_id,
            },
            status_code=400,
        )

    if not holding_id:
        return _error("수정할 보유 내역이 없습니다.")

    if len(holding_id) != len(quantity):
        return _error("보유 ID와 수량 개수가 일치하지 않습니다.")
    if len(set(holding_id)) != len(holding_id):
        return _error("중복된 보유 항목이 포함되어 있습니다.")
    if any(value <= 0 for value in quantity):
        return _error("수량은 0보다 커야 합니다.")

    updates = list(zip(holding_id, quantity))
    try:
        container.holding_repository.bulk_update_by_account(account_id, updates)
    except (ValueError, APIError) as exc:
        return _error(str(exc))

    holdings = container.holding_repository.list_by_account(account_id)
    stocks = container.stock_repository.list_all()
    stock_map = {stock.id: stock for stock in stocks}
    stock_name_map = _build_stock_name_map(container, stocks)
    return templates.TemplateResponse(
        request=request,
        name="accounts/_holdings_bulk_result.html",
        context={
            "success": True,
            "message": "보유 수량을 일괄 저장했습니다.",
            "holdings": holdings,
            "stock_map": stock_map,
            "stock_name_map": stock_name_map,
            "account_id": account_id,
        },
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
    stock_map = {stock.id: stock for stock in stocks}
    stock_name_map = _build_stock_name_map(container, stocks)
    return templates.TemplateResponse(
        request=request,
        name="accounts/_holding_row.html",
        context={
            "holding": holding,
            "stock_map": stock_map,
            "stock_name_map": stock_name_map,
            "account_id": account_id,
        },
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
    stock_map = {stock.id: stock for stock in stocks}
    stock_name_map = _build_stock_name_map(container, stocks)
    return templates.TemplateResponse(
        request=request,
        name="accounts/_holding_form.html",
        context={
            "holding": holding,
            "stock_map": stock_map,
            "stock_name_map": stock_name_map,
            "account_id": account_id,
        },
    )


@router.put("/{account_id}/holdings/{holding_id}", response_class=HTMLResponse)
def update_holding(
    request: Request,
    account_id: UUID,
    holding_id: UUID,
    quantity: Decimal = Form(...),
) -> HTMLResponse:
    container = get_container(request)
    templates = get_templates(request)
    holding = container.holding_repository.update(
        holding_id=holding_id, quantity=quantity
    )
    stocks = container.stock_repository.list_all()
    stock_map = {stock.id: stock for stock in stocks}
    stock_name_map = _build_stock_name_map(container, stocks)
    return templates.TemplateResponse(
        request=request,
        name="accounts/_holding_row.html",
        context={
            "holding": holding,
            "stock_map": stock_map,
            "stock_name_map": stock_name_map,
            "account_id": account_id,
        },
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
        cano = container.kis_cano
        acnt = container.kis_acnt_prdt_cd
        if not cano or not acnt:
            return templates.TemplateResponse(
                request=request,
                name="accounts/_sync_result.html",
                context={
                    "account_id": account_id,
                    "success": False,
                    "message": "KIS 계좌 정보(번호/상품코드)가 설정되지 않았습니다.",
                },
            )
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
