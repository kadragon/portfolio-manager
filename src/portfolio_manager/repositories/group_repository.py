"""Group repository for database operations."""

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

    def create(self, name: str) -> Group:
        """Create a new group.

        Args:
            name: Name of the group.

        Returns:
            Created Group instance.
        """
        response = self.client.table("groups").insert({"name": name}).execute()
        if not response.data or len(response.data) == 0:
            raise ValueError("Failed to create group")
        data = response.data[0]

        return Group(
            id=UUID(str(data["id"])),
            name=str(data["name"]),
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
                created_at=datetime.fromisoformat(str(item["created_at"])),
                updated_at=datetime.fromisoformat(str(item["updated_at"])),
            )
            for item in response.data
        ]
