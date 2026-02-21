"""Web dependency helpers."""

from fastapi import Request
from fastapi.templating import Jinja2Templates

from portfolio_manager.core.container import ServiceContainer


def get_container(request: Request) -> ServiceContainer:
    """Get the service container from app state."""
    return request.app.state.container  # type: ignore[no-any-return]


def get_templates(request: Request) -> Jinja2Templates:
    """Get the Jinja2 templates from app state."""
    return request.app.state.templates  # type: ignore[no-any-return]
