"""Supabase client factory."""

import os
from supabase import Client, create_client


def get_supabase_client() -> Client:
    """Create and return a Supabase client.

    Requires SUPABASE_URL and SUPABASE_KEY environment variables.

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

    return create_client(url, key)
