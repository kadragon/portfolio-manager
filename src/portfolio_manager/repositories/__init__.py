"""Repository pattern for database operations."""

from .group_repository import GroupRepository
from .stock_repository import StockRepository

__all__ = ["GroupRepository", "StockRepository"]
