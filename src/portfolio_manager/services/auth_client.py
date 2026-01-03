from __future__ import annotations

from abc import ABC, abstractmethod
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from portfolio_manager.services.kis_token_store import TokenData


class AuthClient(ABC):
    """Abstract interface for authentication client."""

    @abstractmethod
    def request_access_token(self) -> TokenData:
        """Request a new access token."""
