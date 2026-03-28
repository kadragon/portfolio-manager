"""Group repository for database operations."""

from datetime import datetime, timezone
from uuid import UUID, uuid4

from portfolio_manager.models import Group
from portfolio_manager.services.database import GroupModel


class GroupRepository:
    """Repository for Group database operations."""

    def create(self, name: str, target_percentage: float = 0.0) -> Group:
        """Create a new group."""
        now = datetime.now(timezone.utc)
        row = GroupModel.create(
            id=uuid4(),
            name=name,
            target_percentage=target_percentage,
            created_at=now,
            updated_at=now,
        )
        return self._to_domain(row)

    def list_all(self) -> list[Group]:
        """List all groups."""
        return [self._to_domain(row) for row in GroupModel.select()]

    def delete(self, group_id: UUID) -> None:
        """Delete a group by ID."""
        GroupModel.delete().where(GroupModel.id == group_id).execute()

    def update(
        self,
        group_id: UUID,
        name: str | None = None,
        target_percentage: float | None = None,
    ) -> Group:
        """Update a group by ID."""
        updates: dict = {}
        if name is not None:
            updates["name"] = name
        if target_percentage is not None:
            updates["target_percentage"] = target_percentage

        if not updates:
            raise ValueError("No fields to update")

        updates["updated_at"] = datetime.now(timezone.utc)
        GroupModel.update(updates).where(GroupModel.id == group_id).execute()

        row = GroupModel.get_by_id(group_id)
        return self._to_domain(row)

    @staticmethod
    def _to_domain(row: GroupModel) -> Group:
        return Group(
            id=row.id,
            name=row.name,
            target_percentage=row.target_percentage,
            created_at=row.created_at,
            updated_at=row.updated_at,
        )
