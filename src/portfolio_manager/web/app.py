"""FastAPI web application factory."""

from contextlib import asynccontextmanager
from decimal import Decimal, ROUND_HALF_UP
from pathlib import Path
from typing import AsyncGenerator

from dotenv import load_dotenv
from fastapi import FastAPI
from fastapi.staticfiles import StaticFiles
from fastapi.templating import Jinja2Templates

from portfolio_manager.core.container import ServiceContainer
from portfolio_manager.web.routes import (
    dashboard,
    groups,
    accounts,
    deposits,
    rebalance,
)


@asynccontextmanager
async def lifespan(app: FastAPI) -> AsyncGenerator[None, None]:
    load_dotenv()
    container = ServiceContainer()
    container.setup()
    app.state.container = container
    yield
    container.close()


def _add_filters(templates: Jinja2Templates) -> None:
    def format_krw(value: object) -> str:
        if value is None:
            return "-"
        d = Decimal(str(value))
        return f"₩{d:,.0f}"

    def format_usd(value: object) -> str:
        if value is None:
            return "-"
        d = Decimal(str(value))
        return f"${d:,.2f}"

    def format_percent(value: object) -> str:
        if value is None:
            return "-"
        d = Decimal(str(value))
        formatted = d.quantize(Decimal("0.1"), rounding=ROUND_HALF_UP)
        return f"{formatted}%"

    def format_signed_percent(value: object) -> str:
        if value is None:
            return "-"
        d = Decimal(str(value))
        sign = "+" if d > 0 else ""
        formatted = d.quantize(Decimal("0.1"), rounding=ROUND_HALF_UP)
        return f"{sign}{formatted}%"

    def abs_value(value: object) -> object:
        if value is None:
            return value
        return abs(Decimal(str(value)))  # type: ignore[arg-type]

    templates.env.filters["format_krw"] = format_krw
    templates.env.filters["format_usd"] = format_usd
    templates.env.filters["format_percent"] = format_percent
    templates.env.filters["format_signed_percent"] = format_signed_percent
    templates.env.filters["abs"] = abs_value


def create_app() -> FastAPI:
    templates_dir = Path(__file__).parent / "templates"
    static_dir = Path(__file__).parent / "static"
    templates = Jinja2Templates(directory=str(templates_dir))
    _add_filters(templates)

    app = FastAPI(lifespan=lifespan, title="Portfolio Manager")
    app.state.templates = templates
    app.mount("/static", StaticFiles(directory=str(static_dir)), name="static")

    app.include_router(dashboard.router)
    app.include_router(groups.router)
    app.include_router(accounts.router)
    app.include_router(deposits.router)
    app.include_router(rebalance.router)

    return app


app = create_app()


def run() -> None:
    import uvicorn

    uvicorn.run(
        "portfolio_manager.web.app:app",
        host="127.0.0.1",
        port=8000,
        reload=True,
    )
