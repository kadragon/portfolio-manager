"""Group repository for database operations."""

from typing import Any, cast
from uuid import UUID
from datetime import datetime
from supabase import Client

from portfolio_manager.models import Group


class GroupRepository:
    """Repository for Group database operations."""

    def __init__(self, client: Client):
        """Initialize repository with Supabase client.

        Args:
            client: Supabase client instance.
        """
        self.client = client

    def create(self, name: str, target_percentage: float = 0.0) -> Group:
        """Create a new group.

        Args:
            name: Name of the group.
            target_percentage: Target percentage of the portfolio (0-100).

        Returns:
            Created Group instance.
        """
        response = (
            self.client.table("groups")
            .insert({"name": name, "target_percentage": target_percentage})
            .execute()
        )
        if not response.data or len(response.data) == 0:
            raise ValueError("Failed to create group")
        data = cast(dict[str, Any], response.data[0])

        return Group(
            id=UUID(str(data["id"])),
            name=str(data["name"]),
            target_percentage=float(data.get("target_percentage", 0.0)),
            created_at=datetime.fromisoformat(str(data["created_at"])),
            updated_at=datetime.fromisoformat(str(data["updated_at"])),
        )

    def list_all(self) -> list[Group]:
        """List all groups.

        Returns:
            List of all Group instances.
        """
        response = self.client.table("groups").select("*").execute()
        if not response.data:
            return []

        return [
            Group(
                id=UUID(str(item["id"])),
                name=str(item["name"]),
                target_percentage=float(item.get("target_percentage", 0.0)),
                created_at=datetime.fromisoformat(str(item["created_at"])),
                updated_at=datetime.fromisoformat(str(item["updated_at"])),
            )
            for item in cast(list[dict[str, Any]], response.data)
        ]

    def delete(self, group_id: UUID) -> None:
        """Delete a group by ID.

        Args:
            group_id: ID of the group to delete.
        """
        self.client.table("groups").delete().eq("id", str(group_id)).execute()

    def update(
        self,
        group_id: UUID,
        name: str | None = None,
        target_percentage: float | None = None,
    ) -> Group:
        """Update a group by ID.

        Args:
            group_id: ID of the group to update.
            name: Updated name.
            target_percentage: Updated target percentage.

        Returns:
            Updated Group instance.
        """
        updates: dict[str, Any] = {}
        if name is not None:
            updates["name"] = name
        if target_percentage is not None:
            updates["target_percentage"] = target_percentage

        if not updates:
            raise ValueError("No fields to update")

        response = (
            self.client.table("groups")
            .update(updates)
            .eq("id", str(group_id))
            .execute()
        )
        if not response.data or len(response.data) == 0:
            raise ValueError("Failed to update group")
        data = cast(dict[str, Any], response.data[0])

        return Group(
            id=UUID(str(data["id"])),
            name=str(data["name"]),
            target_percentage=float(data.get("target_percentage", 0.0)),
            created_at=datetime.fromisoformat(str(data["created_at"])),
            updated_at=datetime.fromisoformat(str(data["updated_at"])),
        )
