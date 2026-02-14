"""Supabase project status check and restore utilities."""

from __future__ import annotations

import os
import re
import time
from collections.abc import Callable
from dataclasses import dataclass
from enum import Enum

import httpx


class ProjectStatus(Enum):
    """Supabase project status."""

    ACTIVE = "ACTIVE_HEALTHY"
    PAUSED = "INACTIVE_PAUSED"
    RESTORING = "RESTORING"
    UNKNOWN = "UNKNOWN"


@dataclass
class ProjectCheckResult:
    """Result of project status check."""

    status: ProjectStatus
    restored: bool = False
    error: str | None = None
    is_config_error: bool = False


SUPABASE_API_BASE = "https://api.supabase.com/v1"


def extract_project_ref(supabase_url: str) -> str | None:
    """Extract project ref from Supabase URL.

    Args:
        supabase_url: Full Supabase URL (e.g., https://abc123.supabase.co)

    Returns:
        Project ref string or None if extraction fails.
    """
    match = re.match(r"https://([a-z0-9]+)\.supabase\.co", supabase_url)
    if match:
        return match.group(1)
    return None


def get_project_status(project_ref: str, access_token: str) -> ProjectStatus:
    """Check Supabase project status.

    Args:
        project_ref: Supabase project reference ID
        access_token: Supabase Management API access token

    Returns:
        ProjectStatus enum value.
    """
    headers = {
        "Authorization": f"Bearer {access_token}",
        "Content-Type": "application/json",
    }

    try:
        resp = httpx.get(
            f"{SUPABASE_API_BASE}/projects/{project_ref}",
            headers=headers,
            timeout=30.0,
        )
        if resp.status_code == 200:
            data = resp.json()
            status_str = data.get("status", "")
            if status_str == "ACTIVE_HEALTHY":
                return ProjectStatus.ACTIVE
            elif status_str == "INACTIVE_PAUSED":
                return ProjectStatus.PAUSED
            elif "RESTORING" in status_str:
                return ProjectStatus.RESTORING
        return ProjectStatus.UNKNOWN
    except httpx.HTTPError:
        return ProjectStatus.UNKNOWN


def restore_project(project_ref: str, access_token: str) -> bool:
    """Request project restoration.

    Args:
        project_ref: Supabase project reference ID
        access_token: Supabase Management API access token

    Returns:
        True if restore request was successful.
    """
    headers = {
        "Authorization": f"Bearer {access_token}",
        "Content-Type": "application/json",
    }

    try:
        resp = httpx.post(
            f"{SUPABASE_API_BASE}/projects/{project_ref}/restore",
            headers=headers,
            timeout=30.0,
        )
        return resp.status_code in (200, 201)
    except httpx.HTTPError:
        return False


def wait_for_project_ready(
    project_ref: str,
    access_token: str,
    max_wait_seconds: int = 300,
    poll_interval: int = 10,
) -> bool:
    """Wait for project to become active after restoration.

    Args:
        project_ref: Supabase project reference ID
        access_token: Supabase Management API access token
        max_wait_seconds: Maximum time to wait in seconds
        poll_interval: Time between status checks in seconds

    Returns:
        True if project became active within the wait time.
    """
    elapsed = 0
    while elapsed < max_wait_seconds:
        status = get_project_status(project_ref, access_token)
        if status == ProjectStatus.ACTIVE:
            return True
        time.sleep(poll_interval)
        elapsed += poll_interval
    return False


def check_and_restore_project(
    on_status_update: Callable[[str], None] | None = None,
) -> ProjectCheckResult:
    """Check project status and restore if paused.

    Reads configuration from environment variables:
    - SUPABASE_URL: Project URL to extract project ref
    - SUPABASE_ACCESS_TOKEN: Management API access token

    Args:
        on_status_update: Optional callback for status messages.
                         Signature: (message: str) -> None

    Returns:
        ProjectCheckResult with status and any errors.
    """

    def notify(msg: str) -> None:
        if on_status_update:
            on_status_update(msg)

    supabase_url = os.getenv("SUPABASE_URL")
    access_token = os.getenv("SUPABASE_ACCESS_TOKEN")

    if not supabase_url:
        return ProjectCheckResult(
            status=ProjectStatus.UNKNOWN,
            error="SUPABASE_URL not set",
        )

    if not access_token:
        return ProjectCheckResult(
            status=ProjectStatus.UNKNOWN,
            error="SUPABASE_ACCESS_TOKEN not set (project auto-restore disabled)",
            is_config_error=True,
        )

    project_ref = extract_project_ref(supabase_url)
    if not project_ref:
        return ProjectCheckResult(
            status=ProjectStatus.UNKNOWN,
            error=f"Could not extract project ref from URL: {supabase_url}",
        )

    notify("Checking Supabase project status...")
    status = get_project_status(project_ref, access_token)

    if status == ProjectStatus.ACTIVE:
        return ProjectCheckResult(status=ProjectStatus.ACTIVE)

    if status == ProjectStatus.PAUSED:
        notify("Project is paused. Requesting restoration...")
        if restore_project(project_ref, access_token):
            notify("Restoration requested. Waiting for project to become ready...")
            if wait_for_project_ready(project_ref, access_token):
                notify("Project restored successfully!")
                return ProjectCheckResult(
                    status=ProjectStatus.ACTIVE,
                    restored=True,
                )
            else:
                return ProjectCheckResult(
                    status=ProjectStatus.RESTORING,
                    error="Project restoration in progress but not ready yet. "
                    "Please wait and try again.",
                )
        else:
            return ProjectCheckResult(
                status=ProjectStatus.PAUSED,
                error="Failed to request project restoration. "
                "Please restore manually from Supabase dashboard.",
            )

    if status == ProjectStatus.RESTORING:
        notify("Project restoration in progress. Waiting...")
        if wait_for_project_ready(project_ref, access_token):
            notify("Project is now ready!")
            return ProjectCheckResult(
                status=ProjectStatus.ACTIVE,
                restored=True,
            )
        else:
            return ProjectCheckResult(
                status=ProjectStatus.RESTORING,
                error="Project restoration taking longer than expected. "
                "Please wait and try again.",
            )

    return ProjectCheckResult(
        status=ProjectStatus.UNKNOWN,
        error="Could not determine project status",
    )
