"""Supabase client factory with auto-resume for paused projects."""

import logging
import os
import re
import time
from typing import Callable, TypeVar

import httpx
from supabase import Client, create_client

T = TypeVar("T")

SUPABASE_MANAGEMENT_API_URL = "https://api.supabase.com"

logger = logging.getLogger(__name__)


def extract_project_ref(supabase_url: str) -> str:
    """Extract project ref from Supabase URL.

    Args:
        supabase_url: Full Supabase URL (e.g., https://xxx.supabase.co)

    Returns:
        Project reference string.

    Raises:
        ValueError: If URL format is invalid.
    """
    match = re.match(r"https://([a-z0-9]+)\.supabase\.co", supabase_url)
    if not match:
        raise ValueError(f"Invalid Supabase URL format: {supabase_url}")
    return match.group(1)


def restore_paused_project(project_ref: str, access_token: str) -> bool:
    """Restore a paused Supabase project via Management API.

    Args:
        project_ref: Project reference string.
        access_token: Supabase Personal Access Token.

    Returns:
        True if restore request was successful.
    """
    with httpx.Client(base_url=SUPABASE_MANAGEMENT_API_URL) as client:
        try:
            response = client.post(
                f"/v1/projects/{project_ref}/restore",
                headers={"Authorization": f"Bearer {access_token}"},
            )
        except httpx.HTTPError:
            return False
        return response.status_code == 201


def wait_for_project_ready(
    project_ref: str,
    access_token: str,
    max_wait_seconds: int = 300,
    poll_interval: int = 10,
) -> bool:
    """Wait for project to be ready after restore.

    Args:
        project_ref: Project reference string.
        access_token: Supabase Personal Access Token.
        max_wait_seconds: Maximum time to wait.
        poll_interval: Seconds between status checks.

    Returns:
        True if project is ready, False if timeout.
    """
    with httpx.Client(base_url=SUPABASE_MANAGEMENT_API_URL) as client:
        elapsed = 0
        last_status = ""
        while elapsed < max_wait_seconds:
            response = client.get(
                f"/v1/projects/{project_ref}",
                headers={"Authorization": f"Bearer {access_token}"},
            )
            if response.status_code == 200:
                data = response.json()
                status = data.get("status", "")
                if status != last_status:
                    logger.info("[Supabase] Project status: %s", status)
                    last_status = status
                if status == "ACTIVE_HEALTHY":
                    return True
            else:
                logger.warning(
                    "[Supabase] Failed to get project status (HTTP %d). Retrying...",
                    response.status_code,
                )
            time.sleep(poll_interval)
            elapsed += poll_interval
            logger.debug("[Supabase] Waiting... (%d/%ds)", elapsed, max_wait_seconds)
        return False


def is_paused_project_error(error: Exception) -> bool:
    """Check if the error indicates a paused project.

    Args:
        error: The exception to check.

    Returns:
        True if error suggests paused project.
    """
    if isinstance(error, httpx.ConnectError):
        return True
    error_str = str(error).lower()
    return "nodename nor servname" in error_str or "name resolution" in error_str


def with_auto_resume(func: Callable[[], T]) -> T:
    """Execute function with auto-resume on paused project.

    Args:
        func: Function to execute.

    Returns:
        Result of the function.

    Raises:
        Original exception if resume fails or not applicable.
    """
    try:
        return func()
    except Exception as e:
        if not is_paused_project_error(e):
            raise

        url = os.getenv("SUPABASE_URL")
        access_token = os.getenv("SUPABASE_ACCESS_TOKEN")

        if not url or not access_token:
            raise ValueError(
                "Cannot auto-resume: SUPABASE_ACCESS_TOKEN is not set. "
                "Generate a Personal Access Token at https://supabase.com/dashboard/account/tokens"
            ) from e

        project_ref = extract_project_ref(url)
        logger.info(
            "[Supabase] Project appears paused. Attempting to restore %s...",
            project_ref,
        )

        if not restore_paused_project(project_ref, access_token):
            raise RuntimeError(
                f"Failed to restore Supabase project {project_ref}. "
                "Check your SUPABASE_ACCESS_TOKEN permissions."
            ) from e

        logger.info(
            "[Supabase] Restore request sent. Waiting for project to be ready..."
        )

        if not wait_for_project_ready(project_ref, access_token):
            raise RuntimeError(
                f"Timeout waiting for Supabase project {project_ref} to be ready."
            ) from e

        logger.info("[Supabase] Project restored successfully. Retrying connection...")
        return func()


def get_supabase_client() -> Client:
    """Create and return a Supabase client with auto-resume support.

    Requires SUPABASE_URL and SUPABASE_KEY environment variables.
    Optionally uses SUPABASE_ACCESS_TOKEN for auto-resume of paused projects.

    Returns:
        Supabase client instance.

    Raises:
        ValueError: If required environment variables are not set.
    """
    url = os.getenv("SUPABASE_URL")
    key = os.getenv("SUPABASE_KEY")

    if not url:
        raise ValueError("SUPABASE_URL environment variable is not set")
    if not key:
        raise ValueError("SUPABASE_KEY environment variable is not set")

    def create_and_test_client() -> Client:
        client = create_client(url, key)
        # Test connection by making a simple query
        client.table("groups").select("id").limit(1).execute()
        return client

    return with_auto_resume(create_and_test_client)
