"""Group model."""

from dataclasses import dataclass
from datetime import datetime
from uuid import UUID


@dataclass
class Group:
    """A group of stocks."""

    id: UUID
    name: str
    created_at: datetime
    updated_at: datetime
